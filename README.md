# Ask - AI CLI Client

> A beautiful, lightweight CLI tool for querying AI models directly from your terminal

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-0.2.0-blue)](https://github.com/yourusername/ask/releases)

## ✨ Features

- 🚀 **Simple Usage** - Just type `ask [your question]` 
- 🎨 **Beautiful Output** - Markdown rendering with syntax highlighting
- 🤖 **Multi-Provider** - Supports Gemini, Claude, ChatGPT, and DeepSeek
- 🔄 **Dynamic Models** - Automatically fetches latest models from APIs
- 📋 **Model Discovery** - `--list-models` shows all available models

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/metolius25/ask
cd ask

# Install dependencies
go mod download

# Build
go build -o ask

# Set up config
cp config.yaml.example config.yaml
# Edit config.yaml and add your API key

# Configure your preferred models (optional but recommended)
./ask --configure

# Start using!
./ask What is the meaning of life?
```

## Installation

1. Clone or download this repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Build the binary:
   ```bash
   go build -o ask
   ```
4. (Optional) Move to your PATH:
   ```bash
   sudo mv ask /usr/local/bin/
   ```

## Configuration

Create a `config.yaml` file in one of these locations:
- Current directory: `./config.yaml`
- User config directory: `~/.config/ask/config.yaml`

Use the provided `config.yaml.example` as a template:

```bash
cp config.yaml.example config.yaml
```

Then edit `config.yaml` and add your API key(s):

```yaml
default_provider: gemini

providers:
  gemini:
    api_key: YOUR_ACTUAL_API_KEY_HERE
    # model: gemini-1.5-pro  # Optional: override default model
```

**Note:** The `model` field is optional. If not specified, the app will use the default model for each provider:
- Gemini: `gemini-2.5-flash`
- Claude: `claude-3-5-sonnet-20241022`
- ChatGPT: `gpt-4o`
- DeepSeek: `deepseek-chat`

You can also override the model at runtime with the `-model` flag.

## 🔧 Configuration Wizard

The easiest way to set up your preferred default models is to use the interactive wizard:

```bash
ask --configure
```

This wizard will:
1. **Fetch live models** from each provider's API (if you have API keys configured)
2. **Display available options** with descriptions
3. **Let you choose** your preferred default for each provider
4. **Save automatically** to `~/.config/ask/defaults.yaml`

You can run `ask --configure` anytime to update your preferences. The configuration file looks like:

```yaml
defaults:
  gemini: gemini-2.5-flash
  claude: claude-3-5-sonnet-20241022
  chatgpt: gpt-4o
  deepseek: deepseek-chat
```

**Manual Configuration**: You can also create/edit `~/.config/ask/defaults.yaml` directly using `defaults.yaml.example` as a template.

### Default Model Behavior

- **With configuration**: Uses your chosen defaults from `defaults.yaml`
- **Without configuration**: Uses the first available model from each provider's API
- **Fallback**: If API is unavailable, uses first model from minimal fallback list

This means you never have outdated hardcoded models - everything is dynamic!

### Getting API Keys

- **Gemini**: [Google AI Studio](https://makersuite.google.com/app/apikey)
- **Claude**: [Anthropic Console](https://console.anthropic.com/)
- **ChatGPT**: [OpenAI API Keys](https://platform.openai.com/api-keys)
- **DeepSeek**: [DeepSeek Platform](https://platform.deepseek.com/)

## Usage

### Basic Usage

Simply type `ask` followed by your prompt (no quotes needed):

```bash
ask What is the capital of France?
ask Explain quantum computing in simple terms
ask Write a haiku about programming
```

The response will stream in real-time directly to your terminal.

### Using Different Providers

Override the default provider with the `-provider` flag:

```bash
ask -provider claude Explain machine learning
ask -provider chatgpt Write a Python function
```

### Using Specific Models

Override the model with the `-model` flag:

```bash
ask -model gemini-1.5-pro Explain Einstein's theory of relativity
ask -model gpt-4o-mini Quick summary of Battle of Tannenberg
```

Combine both flags:

```bash
ask -provider claude -model claude-3-opus-20240229 Write a hello world program in Go
```

### List Available Models

See all available models for each provider:

```bash
ask --list-models
```

This fetches the latest models directly from each provider's API (for configured providers) and shows:
- All supported models per provider
- Which model is the default for each
- Model descriptions (where available)

No configuration needed to see default models!

### Help

Get usage information:

```bash
ask -h
# or
ask --help
```

## Troubleshooting

### "Error 404" or "model not found"

The model you're trying to use may have been retired. Run `ask --list-models` to see currently available models, or remove the `model:` line from your config to use the default.

### "API key not configured" or placeholder key detected

Make sure you've:
1. Copied `config.yaml.example` to `config.yaml`
2. Replaced `YOUR_X_API_KEY_HERE` with your actual API key
3. Set the correct `default_provider`

### Empty or blocked responses

Gemini's safety filters may be blocking content. The app uses `HarmBlockOnlyHigh` settings by default. If issues persist, try:
- Rephrasing your prompt
- Using a different provider with `-provider claude` or `-provider chatgpt`

### "config file not found"

The app looks for `config.yaml` in:
1. Current directory (`./config.yaml`)
2. User config directory (`~/.config/ask/config.yaml`)

Create the file in one of these locations.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Future Features

- Interactive chat mode with `-S` flag
- Conversation history
- Custom system prompts
- Token usage tracking

## License

MIT - See [LICENSE](LICENSE) file for details.
