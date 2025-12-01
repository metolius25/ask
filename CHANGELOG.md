# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-12-01

### Added
- **Interactive Configuration Wizard** (`--configure` flag)
  - Fetches available models from provider APIs
  - Lets users choose their preferred default models
  - Saves preferences to `~/.config/ask/defaults.yaml`
  - Can be re-run anytime to update preferences
- `defaults.yaml` user preference file for model selection
- `configure.go` module for configuration wizard
- `defaults.yaml.example` template file

### Changed
- **Removed all hardcoded model defaults** from codebase
- Model selection now fully user-configurable
- Providers use first available model from fallback list if no user preference
- Updated all provider constructors to use dynamic defaults
- `models.go` now loads defaults from user configuration file

### Improved
- Zero maintenance needed when providers release new models
- Users can discover and select latest models via wizard
- Fallback model lists only used as emergency backup
- More flexible and future-proof architecture

## [0.1.0] - 2025-12-01

### Added
- Initial release of Ask CLI
- Support for multiple AI providers (Gemini, Claude, ChatGPT, DeepSeek)
- Real-time streaming responses
- Beautiful markdown rendering with syntax highlighting
- Dynamic model discovery via provider APIs
- Runtime provider and model selection with CLI flags
- Interactive `--list-models` command
- YAML-based configuration with multiple locations support
- First-run helper with setup guidance
- Graceful fallback to hardcoded model lists
- Safety settings configuration for Gemini
- Comprehensive error handling and user-friendly messages

### Features
- Simple usage: `ask [your question]`
- CLI flags: `-provider`, `-model`, `--list-models`
- Config locations: `./config.yaml` or `~/.config/ask/config.yaml`
- Optional model override in config
- Auto-detection of placeholder API keys
- Terminal-optimized markdown rendering

[0.1.0]: https://github.com/yourusername/ask/releases/tag/v0.1.0
