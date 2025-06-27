# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI application template called "mm" that uses the Cobra framework for command-line interfaces and Viper for configuration management. The project serves as a starting point for creating new Go CLI tools with a complete build and release pipeline.

## Development Commands

### Build and Development
- `make build` - Build the binary to `./bin/mm`
- `make run ARGS="arguments"` - Build and run with arguments
- `make fmt` - Format code and tidy modules
- `make test` - Run all tests
- `make clean` - Remove build artifacts
- `make all` - Full development cycle (clean, format, build, test)

### Installation
- `make install` - Build and install to `$GOPATH/bin`

### Configuration Management
- `./bin/mm config init` - Initialize configuration file
- `./bin/mm config view` - View current configuration

## Project Architecture

### Core Structure
- **main.go**: Entry point that calls `cmd.Execute()`
- **cmd/root.go**: Root command definition with Cobra, contains CLI_NAME constant and global configuration loading
- **cmd/config.go**: Configuration management subcommands
- **cmd/version.go**: Version information command
- **internal/config/config.go**: Configuration loading/saving logic using Viper

### Key Patterns
- **CLI Name**: Defined as `CLI_NAME = "mm"` in `cmd/root.go:12` - change this to customize the binary name
- **Version Info**: Build-time injection via ldflags in Makefile and GoReleaser
- **Config System**: Supports YAML config files in `~/.config/mm/` with environment variable override
- **Cobra Integration**: Uses persistent flags and PreRunE hooks for config loading

### Configuration System
- Default config location: `~/.config/mm/.mm.yaml`
- Environment variables prefixed with CLI_NAME (e.g., `MM_EXAMPLE`)
- Auto-creates config directory structure
- Falls back to default values if config file missing

## Build System

### Local Development
- Uses Go modules (`go.mod`)
- Makefile provides development tasks
- Binary name controlled by `BINARY_NAME` in Makefile:4
- Version and build time injected via ldflags

### Release Pipeline
- **GoReleaser**: Multi-platform builds (Linux, macOS, Windows)
- **GitHub Actions**: Automated releases on git tags
- **Homebrew Integration**: Automatic formula updates via `make update-homebrew`

### Cross-Platform Support
- Supports: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- Archives as tar.gz (zip for Windows)
- Checksums automatically generated

## Customization Notes

This is a template project. To adapt for a new CLI:

1. Change `CLI_NAME` in `cmd/root.go:12`
2. Update module name in `go.mod:1`
3. Update `BINARY_NAME` and ldflags paths in `Makefile`
4. Update paths in `.goreleaser.yaml:19`
5. Run `./customize.sh github.com/yourusername/yourproject yourcli` for automated replacement

## Important Files for Modification

- **cmd/root.go:23-27**: Main command description and help text
- **internal/config/config.go:11-17**: Configuration structure definition
- **Makefile:4,8**: Binary name and build configuration
- **.goreleaser.yaml:19**: Release build configuration