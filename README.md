# CLI Template

A template project for quickly creating Go CLI applications.

## Features

- Command-line framework based on [Cobra](https://github.com/spf13/cobra)
- Configuration management using [Viper](https://github.com/spf13/viper)
- Cross-platform support (Linux, macOS, Windows)
- Complete build system (Makefile + GoReleaser)
- Built-in help and version commands
- Automatic Homebrew release and update support
- GitHub Actions CI/CD workflow

## Quick Start

### Build the Project

```bash
# Format code
make fmt

# Build the project
make build

# Run the project
make run

# Show help
make help
```

### Use the Binary

```bash
# Run the compiled binary directly
./bin/mycli

# Show help
./bin/mycli --help

# Show version
./bin/mycli version

# View config
./bin/mycli config view

# Initialize config
./bin/mycli config init
```

## How to Adapt for Your Own CLI Project

Follow these steps to quickly convert this template into your own CLI project:

1. **Change CLI Name**:

   Edit the `cmd/root.go` file and change the value of the `CLI_NAME` variable to your CLI name:

   ```go
   // CLI_NAME is the name of the CLI command
   CLI_NAME = "your-command-name"
   ```

2. **Change Go Module Name**:

   Change the module name in the `go.mod` file:

   ```
   module github.com/your-username/your-project-name
   ```

3. **Update Import Paths**:

   Search for `github.com/samzong/cli-template` in all files and replace it with your new module path.

4. **Update Makefile**:

   Edit the `Makefile` and update `BINARY_NAME` and the module path in `LDFLAGS`:

   ```makefile
   BINARY_NAME=your-command-name
   LDFLAGS=-ldflags "-X github.com/your-username/your-project-name/cmd.Version=$(VERSION) -X 'github.com/your-username/your-project-name/cmd.BuildTime=$(BUILDTIME)'"
   ```

5. **Update .goreleaser.yaml**:

   Edit the module path in the `.goreleaser.yaml` file:

   ```yaml
   ldflags:
     - -s -w -X github.com/your-username/your-project-name/cmd.Version={{.Version}} -X github.com/your-username/your-project-name/cmd.BuildTime={{.Date}}
   ```

6. **Update Homebrew Related Settings**:

   Edit the Homebrew related variables in the `Makefile`:

   ```makefile
   HOMEBREW_TAP_REPO=homebrew-tap
   FORMULA_FILE=Formula/$(BINARY_NAME).rb
   ```

   Adjust the GitHub username in the `update-homebrew` task:

   ```makefile
   @cd tmp && git clone https://$(GH_PAT)@github.com/your-username/$(HOMEBREW_TAP_REPO).git
   ```

7. **Update LICENSE and README**:

   Edit the copyright information in the `LICENSE` file and the `README.md` file.

### One-Click Replacement Tool

Use the `customize.sh` script in the project root directory to automatically perform the above replacements:

```bash
chmod +x customize.sh
./customize.sh github.com/yourusername/yourproject yourcli
```

## Release Process

This template includes a complete GitHub Actions release workflow:

1. **Create a Version Tag**:

   ```bash
   git tag -a v0.1.0 -m "First release"
   git push origin v0.1.0
   ```

2. **Automatic Build and Release**:

   After pushing the tag, GitHub Actions will automatically:

   - Build binaries for all platforms
   - Create a GitHub Release
   - Upload build artifacts
   - Trigger the Homebrew update process

3. **Update Homebrew Formula**:

   The Homebrew update workflow will:

   - Download the released binaries
   - Calculate SHA256 checksums
   - Update the Homebrew formula
   - Create a PR to your Homebrew Tap repository

### Required Secrets

To make the release process work, you need to set the following secrets in your GitHub repository:

- `GH_PAT`: A GitHub Personal Access Token with sufficient permissions to create PRs and update the Homebrew formula

## License

MIT License
