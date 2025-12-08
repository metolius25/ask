package main

import (
	"fmt"
	"os/user"
	"strings"
	"sync"

	"ask/provider"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// Layout constants
const (
	headerHeight = 3
	inputHeight  = 3
)

// Styles
var (
	primaryColor   = lipgloss.Color("#00D7FF")
	secondaryColor = lipgloss.Color("#00FF87")
	mutedColor     = lipgloss.Color("#666666")
	errorColor     = lipgloss.Color("#FF5555")

	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	subtitleStyle = lipgloss.NewStyle().Foreground(mutedColor)
	userStyle     = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	modelStyle    = lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	sepStyle      = lipgloss.NewStyle().Foreground(mutedColor)
	helpStyle     = lipgloss.NewStyle().Foreground(mutedColor)
)

// Message types
type ChatMessage struct {
	Role    string
	Content string
}

type streamChunkMsg string
type streamDoneMsg struct{ err error }

// streamWriter implements io.Writer and sends chunks to the program
type streamWriter struct {
	prog *tea.Program
}

func (w *streamWriter) Write(p []byte) (n int, err error) {
	w.prog.Send(streamChunkMsg(string(p)))
	return len(p), nil
}

// SessionModel is the main TUI model
type SessionModel struct {
	provider      provider.Provider
	providerName  string
	modelName     string
	username      string
	messages      []ChatMessage
	currentStream string
	textarea      textarea.Model
	viewport      viewport.Model
	width         int
	height        int
	loading       bool
	ready         bool
	prog          *tea.Program
	streamMu      sync.Mutex
}

// NewSessionModel creates a new session
func NewSessionModel(p provider.Provider, providerName, modelName string) *SessionModel {
	username := "you"
	if u, err := user.Current(); err == nil && u.Username != "" {
		username = u.Username
	}

	ta := textarea.New()
	ta.Placeholder = "Type message... (Enter to send)"
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	return &SessionModel{
		provider:     p,
		providerName: providerName,
		modelName:    modelName,
		username:     username,
		textarea:     ta,
		messages:     []ChatMessage{},
	}
}

func (m *SessionModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}

func (m *SessionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}

			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			// Handle commands
			if strings.HasPrefix(input, "/") {
				return m.handleCommand(input)
			}

			// Add user message
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			m.textarea.Reset()
			m.loading = true
			m.currentStream = ""
			m.refreshViewport()

			return m, m.startStreaming(input)

		case tea.KeyUp, tea.KeyDown:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		vpHeight := msg.Height - headerHeight - inputHeight - 2
		if vpHeight < 1 {
			vpHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.MouseWheelEnabled = false
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}

		m.textarea.SetWidth(msg.Width - 2)
		m.refreshViewport()
		return m, nil

	case streamChunkMsg:
		m.streamMu.Lock()
		m.currentStream += string(msg)
		m.streamMu.Unlock()
		m.refreshViewport()
		return m, nil

	case streamDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.messages = append(m.messages, ChatMessage{
				Role:    "error",
				Content: fmt.Sprintf("Error: %v", msg.err),
			})
		} else {
			m.streamMu.Lock()
			content := m.currentStream
			m.streamMu.Unlock()
			m.messages = append(m.messages, ChatMessage{
				Role:    "assistant",
				Content: content,
			})
		}
		m.currentStream = ""
		m.refreshViewport()
		return m, nil
	}

	// Update textarea
	var taCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)

	return m, taCmd
}

func (m *SessionModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Header
	header := headerStyle.Render(m.modelName) + "\n"
	header += subtitleStyle.Render("Session Mode") + "\n"
	header += sepStyle.Render(strings.Repeat("─", m.width))

	// Viewport
	chatView := m.viewport.View()

	// Input area
	inputSep := sepStyle.Render(strings.Repeat("─", m.width))
	input := m.textarea.View()

	// Status
	status := helpStyle.Render("Ctrl+C: exit | /help: commands")

	return header + "\n" + chatView + "\n" + inputSep + "\n" + input + "\n" + status
}

func (m *SessionModel) refreshViewport() {
	var sb strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(userStyle.Render(m.username+" >") + "\n")
			sb.WriteString(msg.Content + "\n\n")

		case "assistant":
			sb.WriteString(modelStyle.Render(m.modelName+" >") + "\n")
			r, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(m.width-4),
			)
			rendered, err := r.Render(msg.Content)
			if err != nil {
				sb.WriteString(msg.Content)
			} else {
				sb.WriteString(strings.TrimSpace(rendered))
			}
			sb.WriteString("\n\n")

		case "system":
			sb.WriteString(helpStyle.Render(msg.Content) + "\n\n")

		case "error":
			errStyle := lipgloss.NewStyle().Foreground(errorColor)
			sb.WriteString(errStyle.Render(msg.Content) + "\n\n")
		}
	}

	// Show current streaming content
	if m.loading {
		sb.WriteString(modelStyle.Render(m.modelName+" >") + "\n")
		m.streamMu.Lock()
		content := m.currentStream
		m.streamMu.Unlock()
		if content == "" {
			sb.WriteString(lipgloss.NewStyle().Italic(true).Foreground(mutedColor).Render("Thinking..."))
		} else {
			// Wrap content to ensure viewport scrolls correctly
			wrapped := lipgloss.NewStyle().Width(m.width - 4).Render(content)
			sb.WriteString(wrapped)
		}
		sb.WriteString("\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m *SessionModel) handleCommand(input string) (tea.Model, tea.Cmd) {
	cmd := strings.ToLower(strings.TrimSpace(input))

	switch cmd {
	case "/exit", "/quit", "/q":
		return m, tea.Quit

	case "/clear", "/c":
		m.messages = []ChatMessage{}
		m.messages = append(m.messages, ChatMessage{
			Role:    "system",
			Content: "Conversation cleared.",
		})
		m.refreshViewport()
		return m, nil

	case "/help", "/h", "/?":
		helpText := `Commands:
  /help, /h   - Show this help
  /clear, /c  - Clear conversation
  /exit, /q   - Exit session`
		m.messages = append(m.messages, ChatMessage{
			Role:    "system",
			Content: helpText,
		})
		m.refreshViewport()
		return m, nil

	default:
		m.messages = append(m.messages, ChatMessage{
			Role:    "error",
			Content: fmt.Sprintf("Unknown command: %s", cmd),
		})
		m.refreshViewport()
		return m, nil
	}
}

func (m *SessionModel) startStreaming(input string) tea.Cmd {
	return func() tea.Msg {
		var msgs []provider.Message
		for _, msg := range m.messages {
			if msg.Role == "user" || msg.Role == "assistant" {
				msgs = append(msgs, provider.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}

		// Create a writer that sends chunks to the program
		writer := &streamWriter{prog: m.prog}

		err := m.provider.QueryStreamWithHistory(msgs, writer)
		return streamDoneMsg{err: err}
	}
}

// RunSessionTUI starts the TUI session
func RunSessionTUI(p provider.Provider, providerName, modelName string) error {
	model := NewSessionModel(p, providerName, modelName)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	model.prog = prog
	_, err := prog.Run()
	return err
}
