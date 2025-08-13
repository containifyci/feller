# Feller

Feller is a lightweight secret management tool optimized for GitHub Actions environments. It can parse Teller configuration files and handle secrets efficiently in GitHub Actions workflows, with automatic fallback to the original Teller binary when not running in GitHub Actions.

## Features

- **GitHub Actions Optimized**: Automatically detects GitHub Actions environment
- **Teller Compatible**: Uses existing `.teller.yml` configuration files
- **Provider Support**: Supports Google Secret Manager (via environment variables) and dotenv providers
- **Multiple Export Formats**: JSON, YAML, ENV, CSV, and shell export formats
- **Automatic Fallback**: Falls back to original `teller` binary when not in GitHub Actions

## Installation

```bash
go build -o feller .
```

## Usage

Feller uses the same command syntax as Teller:

### Debug and Verbose Modes

```bash
# Enable verbose output
feller --verbose run -- ./deploy.sh

# Enable detailed debug logging
feller --debug run -- ./deploy.sh

# Combine both for maximum verbosity
feller --verbose --debug export json
```

### Missing Environment Variable Handling

By default, Feller fails with a helpful error when required environment variables are missing in GitHub Actions:

```bash
# This will fail with detailed error message if secrets are missing
feller run -- ./deploy.sh

# Use --silent flag to continue with only available secrets (not recommended)
feller --silent run -- ./deploy.sh
```

The error message will show exactly which environment variables are missing and provide the correct GitHub Actions workflow syntax to add them.

### Running Commands with Secrets

```bash
# Run a command with secrets injected as environment variables
feller run -- ./deploy.sh

# Run with shell support
feller run --shell -- "echo $DATABASE_URL | head -c 10"

# Reset environment before running
feller run --reset -- node app.js
```

### Exporting Secrets

```bash
# Export as environment variables
feller env

# Export as JSON
feller export json

# Export as YAML
feller export yaml

# Export for shell evaluation
eval "$(feller sh)"
```

## Configuration

Feller uses standard `.teller.yml` configuration files. It currently supports:

### Google Secret Manager Provider
When running in GitHub Actions, GSM providers read from environment variables:

```yaml
providers:
  gha_secrets:
    kind: google_secretmanager
    maps:
      - id: app_secrets
        keys:
          DATABASE_URL: DATABASE_URL  # Read $DATABASE_URL, output as DATABASE_URL
          API_KEY: API_KEY            # Read $API_KEY, output as API_KEY
          REDIS_PASSWORD: REDIS_PASS  # Read $REDIS_PASSWORD, output as REDIS_PASS
```

### Dotenv Provider
Reads secrets from `.env` files:

```yaml
providers:
  local_config:
    kind: dotenv
    maps:
      - id: local_vars
        path: .env.production
        keys:
          DB_PASSWORD: DATABASE_PASSWORD  # Read DB_PASSWORD from file, output as DATABASE_PASSWORD
```

## GitHub Actions Integration

### Typical Workflow

```yaml
name: Deploy
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Feller
        run: |
          # Install feller binary
          curl -L https://github.com/yourorg/feller/releases/download/v1.0.0/feller -o feller
          chmod +x feller
          
      - name: Deploy with secrets
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
          API_KEY: ${{ secrets.API_KEY }}
          REDIS_PASSWORD: ${{ secrets.REDIS_PASSWORD }}
        run: feller run -- ./deploy.sh
```

### Configuration Example

```yaml
# .teller.yml
providers:
  # GitHub Actions secrets (read from environment)
  gha_secrets:
    kind: google_secretmanager
    maps:
      - id: app_secrets
        keys:
          DATABASE_URL: DATABASE_URL
          API_KEY: API_KEY
          REDIS_PASSWORD: REDIS_PASS

  # Local development (read from .env file)
  local_dev:
    kind: dotenv
    maps:
      - id: dev_vars
        path: .env.local
        keys:
          DEBUG_MODE: DEBUG
          LOG_LEVEL: LOG_LEVEL
```

## Behavior

- **In GitHub Actions**: Feller handles secret collection and command execution
- **Outside GitHub Actions**: Feller automatically falls back to the original `teller` binary
- **Configuration**: Uses the same `.teller.yml` files as Teller
- **Commands**: Supports `run`, `export`, `env`, and `sh` commands

## Requirements

- Go 1.21 or later (for building)
- GitHub Actions environment (for GitHub Actions mode)
- Original `teller` binary in PATH (for fallback mode)

## Supported Providers

- `google_secretmanager`: Reads from environment variables in GitHub Actions
- `dotenv`: Reads from `.env` files on filesystem

## Commands

- `feller run -- command`: Execute command with secrets as environment variables
- `feller export [format]`: Export secrets in specified format (json, yaml, env, csv)
- `feller env`: Export secrets in environment variable format
- `feller sh`: Export secrets as shell export statements