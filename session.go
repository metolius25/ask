package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"sync"
	"syscall"
	"time"

	"ask/provider"

	"github.com/charmbracelet/glamour"
)

// ANSI codes
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	cyan      = "\033[36m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	magenta   = "\033[35m"
	gray      = "\033[90m"
	bgGray    = "\033[48;5;236m"
	clearLine = "\033[2K\r"
)

// Spinner frames (braille dots)
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Session holds the conversation state
type Session struct {
	provider     provider.Provider
	providerName string
	modelName    string
	username     string
	messages     []provider.Message
	mu           sync.Mutex
}

// RunSessionREPL starts an interactive session
func RunSessionREPL(p provider.Provider, providerName, modelName string) error {
	// Get system username
	username := "you"
	if u, err := user.Current(); err == nil && u.Username != "" {
		username = u.Username
	}

	session := &Session{
		provider:     p,
		providerName: providerName,
		modelName:    modelName,
		username:     username,
		messages:     []provider.Message{},
	}

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n\n%s👋 Goodbye!%s\n\n", yellow, reset)
		os.Exit(0)
	}()

	// Print header
	session.printHeader()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Prompt with username
		prompt := fmt.Sprintf("%s%s%s › %s", bold, cyan, session.username, reset)
		fmt.Print(prompt)
		os.Stdout.Sync() // Force flush to ensure visibility before blocking

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if session.handleCommand(input) {
				break // exit requested
			}
			continue
		}

		// Add user message
		session.mu.Lock()
		session.messages = append(session.messages, provider.Message{
			Role:    "user",
			Content: input,
		})
		msgs := make([]provider.Message, len(session.messages))
		copy(msgs, session.messages)
		session.mu.Unlock()

		// Query with spinner
		response, err := session.queryWithSpinner(msgs)
		if err != nil {
			errStr := err.Error()
			// Detect model not found errors (404, invalid model, etc.)
			if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") ||
				strings.Contains(errStr, "does not exist") || strings.Contains(errStr, "Invalid model") ||
				strings.Contains(errStr, "invalid_model") {
				fmt.Printf("\n%s✗ Model '%s' not found%s\n", red, session.modelName, reset)
				fmt.Printf("%s  Use 'ask --list-models' to see available models, or /model %s for default%s\n", dim, session.providerName, reset)
			} else {
				fmt.Printf("\n%s✗ Error: %v%s\n", red, err, reset)
			}
			// Remove failed message
			session.mu.Lock()
			if len(session.messages) > 0 {
				session.messages = session.messages[:len(session.messages)-1]
			}
			session.mu.Unlock()
			continue
		}

		// Add assistant response
		session.mu.Lock()
		session.messages = append(session.messages, provider.Message{
			Role:    "assistant",
			Content: response,
		})
		session.mu.Unlock()

		// Assistant "prompt" (model name)
		fmt.Printf("\n%s%s%s › %s\n", bold, green, session.modelName, reset)
		renderMarkdownToTerminal(response)

		// Add spacing before next user prompt
		fmt.Println()
		fmt.Println()
	}

	return scanner.Err()
}

func (s *Session) printHeader() {
	// Calculate box width based on model name (minimum 45, max 60)
	modelLen := len(s.modelName)
	boxWidth := 45
	if modelLen+6 > boxWidth {
		boxWidth = modelLen + 6
		if boxWidth > 60 {
			boxWidth = 60
		}
	}

	// Truncate model name if too long
	displayName := s.modelName
	if len(displayName) > boxWidth-6 {
		displayName = displayName[:boxWidth-9] + "..."
	}

	innerWidth := boxWidth - 2

	fmt.Println()
	fmt.Printf("%s╭%s╮%s\n", gray, strings.Repeat("─", innerWidth), reset)
	fmt.Printf("%s│%s  %s%s%s%s%s│%s\n", gray, reset, bold+magenta, displayName, reset, strings.Repeat(" ", innerWidth-len(displayName)-2), gray, reset)
	fmt.Printf("%s│%s  %sSession Mode%s%s%s│%s\n", gray, reset, dim, reset, strings.Repeat(" ", innerWidth-14), gray, reset)
	fmt.Printf("%s╰%s╯%s\n", gray, strings.Repeat("─", innerWidth), reset)
	fmt.Printf("\n%s  /help • /model • /clear • /exit%s\n", dim, reset)
}

func (s *Session) queryWithSpinner(msgs []provider.Message) (string, error) {
	type result struct {
		response string
		err      error
	}
	resultChan := make(chan result, 1)

	// Start query in goroutine
	go func() {
		var buf strings.Builder
		err := s.provider.QueryStreamWithHistory(msgs, &buf)
		resultChan <- result{response: buf.String(), err: err}
	}()

	// Spinner animation
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-done:
				fmt.Print(clearLine)
				return
			default:
				frame := spinnerFrames[i%len(spinnerFrames)]
				fmt.Printf("\r%s%s %sThinking...%s", yellow, frame, dim, reset)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	// Wait for result
	res := <-resultChan
	close(done)
	wg.Wait() // Deterministically wait for spinner to clear line

	return res.response, res.err
}

func (s *Session) handleCommand(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/exit", "/quit", "/q":
		fmt.Printf("\n%s👋 Goodbye!%s\n\n", yellow, reset)
		return true

	case "/clear", "/c":
		s.mu.Lock()
		s.messages = []provider.Message{}
		s.mu.Unlock()
		// Clear screen and reprint header
		fmt.Print("\033[2J\033[H") // clear screen, move cursor to top
		s.printHeader()
		fmt.Printf("%s✓ Conversation cleared%s\n", dim, reset)

	case "/model", "/m":
		if len(parts) < 2 {
			fmt.Printf("\n%sUsage: /model <name> or /model <provider> or /model <provider/model>%s\n", dim, reset)
			return false
		}

		modelSpec := parts[1]
		newProvider, newModel := ParseModelSpec(modelSpec)

		// If no provider detected, try to resolve from model name
		if newProvider == "" {
			newProvider = ResolveProviderFromModel(newModel)
		}

		// If still no provider but the spec matches a known provider name, use it
		knownProviders := []string{"gemini", "claude", "chatgpt", "deepseek", "mistral", "qwen"}
		for _, kp := range knownProviders {
			if newModel == kp {
				newProvider = kp
				newModel = "" // Will use default
				break
			}
		}

		if newProvider == "" {
			newProvider = s.providerName
		}

		config, err := LoadConfigSafe()
		if err != nil {
			fmt.Printf("\n%s✗ Error loading config: %v%s\n", red, err, reset)
			return false
		}

		pc, exists := config.Providers[newProvider]
		if !exists {
			fmt.Printf("\n%s✗ Provider '%s' not configured%s\n", red, newProvider, reset)
			return false
		}

		// If no model specified, use provider's default from config or fallback
		if newModel == "" {
			if pc.Model != "" {
				newModel = pc.Model
			} else if dm, ok := defaultModels[newProvider]; ok {
				newModel = dm
			}
		}

		newProviderInstance := createProvider(newProvider, pc.APIKey, newModel)
		if newProviderInstance == nil {
			fmt.Printf("\n%s✗ Unknown provider: %s%s\n", red, newProvider, reset)
			return false
		}

		s.provider = newProviderInstance
		s.providerName = newProvider
		s.modelName = newModel
		fmt.Printf("\n%s✓ Switched to %s/%s%s\n", green, newProvider, newModel, reset)

	case "/help", "/h", "/?":
		fmt.Printf("\n%s", dim)
		fmt.Println("  Commands:")
		fmt.Println("    /help, /h    Show this help")
		fmt.Println("    /model, /m   Switch model (e.g., /model gpt-4o)")
		fmt.Println("    /clear, /c   Clear conversation history")
		fmt.Println("    /exit, /q    Exit session")
		fmt.Printf("%s\n", reset)

	default:
		fmt.Printf("\n%s✗ Unknown command: %s%s\n", red, cmd, reset)
	}

	return false
}

func renderMarkdownToTerminal(content string) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		fmt.Println(content)
		fmt.Println()
		return
	}

	out, err := r.Render(content)
	if err != nil {
		fmt.Println(content)
		fmt.Println()
		return
	}

	// Ensure output ends cleanly and leaves a blank line
	fmt.Println(strings.TrimSpace(out))
	fmt.Println()
}
