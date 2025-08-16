# Setup Feller Action

A GitHub Action to download and setup the [Feller](https://github.com/containifyci/feller) binary for secret management in GitHub Actions workflows.

Feller is a lightweight secret management tool optimized for GitHub Actions environments that can parse Teller configuration files and handle secrets efficiently.

> **Note**: This action provides seamless integration for Feller in GitHub workflows.

## Features

- **Cross-platform support**: Works on Linux, macOS, and Windows runners
- **Version flexibility**: Install latest version or specify exact version
- **Binary caching**: Automatically caches downloaded binaries for faster subsequent runs
- **PATH integration**: Adds feller to PATH for use in subsequent workflow steps
- **Command execution**: Optionally run feller commands directly through the action
- **Error handling**: Comprehensive error messages and validation

## Usage

### Basic Installation

Install the latest version of feller:

```yaml
- name: Setup Feller
  uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
```

### Install Specific Version

```yaml
- name: Setup Feller
  uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
  with:
    version: 'v0.0.1'
```

### Install and Run Command

```yaml
- name: Setup Feller and Run
  uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
  with:
    command: 'run'
    args: '-- ./deploy.sh'
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
    API_KEY: ${{ secrets.API_KEY }}
```

### Multi-step Usage

Install once and use in multiple steps:

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Feller
        uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
        
      - name: Export secrets as environment file
        run: feller export env > .env.production
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
          API_KEY: ${{ secrets.API_KEY }}
          
      - name: Run deployment script
        run: feller run -- ./deploy.sh
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
          API_KEY: ${{ secrets.API_KEY }}
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `version` | Version of Feller to install | No | `latest` |
| `command` | Feller command to run (e.g., `run`, `export`, `env`) | No | |
| `args` | Arguments to pass to the feller command | No | |
| `working-directory` | Working directory to run feller from | No | `.` |

## Outputs

| Output | Description |
|--------|-------------|
| `feller-path` | Path to the installed feller binary |
| `version-installed` | Version of feller that was installed |

## Supported Platforms

| OS | Architecture | Supported |
|----|--------------|-----------|
| Linux | x64 (amd64) | ✅ |
| Linux | ARM64 | ✅ |
| macOS | x64 (amd64) | ✅ |
| macOS | ARM64 (Apple Silicon) | ✅ |
| Windows | x64 (amd64) | ✅ |
| Windows | ARM64 | ❌ |

## Examples

### Complete Deployment Workflow

```yaml
name: Deploy Application

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        
      - name: Setup Feller
        uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
        id: feller
        
      - name: Deploy with secrets
        run: feller run -- ./scripts/deploy.sh
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
          API_KEY: ${{ secrets.API_KEY }}
          REDIS_PASSWORD: ${{ secrets.REDIS_PASSWORD }}
          
      - name: Verify deployment
        run: |
          echo "Feller version: ${{ steps.feller.outputs.version-installed }}"
          feller env | grep -c "=" # Count exported variables
```

### Export Secrets for Later Use

```yaml
- name: Setup Feller and export secrets
  uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
  with:
    command: 'export'
    args: 'json'
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
    API_KEY: ${{ secrets.API_KEY }}
```

### Cross-platform Matrix Build

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]

jobs:
  test:
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Feller
        uses: containifyci/feller/.github/actions/setup-feller@v0.0.1
        
      - name: Test feller installation
        run: feller --help
```

## Feller Configuration

This action works with standard `.teller.yml` configuration files. Place your configuration in the repository root:

```yaml
# .teller.yml
providers:
  github_secrets:
    kind: google_secretmanager
    maps:
      - id: app_secrets
        keys:
          DATABASE_URL: DATABASE_URL
          API_KEY: API_KEY
          REDIS_PASSWORD: REDIS_PASS
```

## Troubleshooting

### Binary Download Fails

If the binary download fails, check:
- The specified version exists in [releases](https://github.com/containifyci/feller/releases)
- Your runner has internet access
- GitHub API rate limits (unlikely but possible)

### Binary Not Found in PATH

The action automatically adds feller to PATH. If you encounter issues:
- Use the full path from the `feller-path` output
- Check that the installation step completed successfully

### Command Execution Fails

When using the `command` input:
- Ensure the command is valid (check `feller --help`)
- Verify required environment variables are set
- Check the `.teller.yml` configuration file exists and is valid

### Platform Not Supported

Currently supported platforms:
- Linux (x64, ARM64)
- macOS (x64, ARM64) 
- Windows (x64 only)

Windows ARM64 is not supported by feller releases.

## License

This action is provided under the same license as the Feller project.

## Contributing

Issues and contributions are welcome! Please check the main [Feller repository](https://github.com/containifyci/feller) for guidelines.