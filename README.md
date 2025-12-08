# Ask - AI CLI Client

> A beautiful, lightweight CLI tool for querying AI models directly from your terminal

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![GitHub Release](https://img.shields.io/github/v/release/metolius25/ask)](https://github.com/metolius25/ask/releases)
[![Build](https://github.com/metolius25/ask/actions/workflows/release.yml/badge.svg)](https://github.com/metolius25/ask/actions)

## Features

- 🚀 **Simple Usage** - Just type `ask [your question]`
- 🎨 **Beautiful Output** - Markdown rendering with syntax highlighting
- 🤖 **Multi-Provider** - Gemini, Claude, ChatGPT, DeepSeek, Mistral, Qwen
- 🔄 **Smart Detection** - Auto-detects provider from model name
- 📋 **Profiles** - Save favorite configs with `-P fast`
- 💬 **Interactive Sessions** - Multi-turn conversations with `-s`
- ⚡ **Short Flags** - `-m`, `-p`, `-s`, `-P` for quick usage

## Quick Start

```bash
# Clone and build
git clone https://github.com/metolius25/ask
cd ask && go build -o ask

# First run - interactive setup
./ask

# Or manually configure
cp config.yaml.example ~/.config/ask/config.yaml
# Edit and add your API key

# Start using!
ask What is the meaning of life?
```

## Usage

```bash
# Basic query
ask What is quantum computing?

# Use a specific model (auto-detects provider)
ask -m gpt-4o Explain neural networks

# Use provider/model syntax
ask -m claude/claude-3-opus Write a poem

# Use a profile
ask -P fast Quick summary of relativity

# Interactive session
ask -s

# List available models
ask --list-models

# Configure defaults
ask --config
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `-model` | `-m` | Model to use (e.g., `gpt-4o`, `gemini/gemini-2.5-pro`) |
| `-provider` | `-p` | Provider (gemini, claude, chatgpt, deepseek, mistral, qwen) |
| `-profile` | `-P` | Use a named profile from config |
| `-session` | `-s` | Start interactive session mode |
| `-version` | `-v` | Show version |
| `--list-models` | | List available models |
| `--config` | | Configure API keys (`--config` or `--config qwen`) |

## Configuration

Config file: `~/.config/ask/config.yaml`

```yaml
default_provider: gemini

providers:
  gemini:
    api_key: YOUR_API_KEY
    model: gemini-2.5-flash  # optional default
  claude:
    api_key: YOUR_API_KEY

# Optional: quick-switch profiles
profiles:
  fast: gemini/gemini-2.5-flash
  smart: claude/claude-3-opus-20240229
  cheap: deepseek/deepseek-chat
```

### Getting API Keys

- **Gemini**: [Google AI Studio](https://makersuite.google.com/app/apikey)
- **Claude**: [Anthropic Console](https://console.anthropic.com/)
- **ChatGPT**: [OpenAI API Keys](https://platform.openai.com/api-keys)
- **DeepSeek**: [DeepSeek Platform](https://platform.deepseek.com/)
- **Mistral**: [Mistral Console](https://console.mistral.ai/)
- **Qwen**: [Alibaba DashScope](https://dashscope.console.aliyun.com/)

## Session Mode

Start an interactive session:

```bash
ask -s
```

Session commands:
- `/model <name>` - Switch model (e.g., `/model gpt-4o`)
- `/clear` - Clear conversation
- `/help` - Show commands
- `/exit` - Exit session

## Troubleshooting

**"Provider not configured"** - Add API key to config.yaml

**"Model not found"** - Run `ask --list-models` to see available models

**First time?** - Just run `ask` and follow the interactive setup

## License

MIT - See [LICENSE](LICENSE) file for details.
