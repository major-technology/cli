# Major CLI

`major` is Major on the command line. It brings authentication, app creation, and app deployment directly to your own development environment so that you can create internal apps with ease.

## Documentation

Major CLI is supported for users on macOS, Windows, and Linux.

## Installation

`major` is available via Homebrew:

```bash
brew tap major-technology/tap
brew install major-technology/tap/major
```

### Build from source

Prerequisites:
- Go 1.24 or higher

1. Clone the repository:
```bash
git clone https://github.com/major-technology/cli.git
cd cli
```

2. Build the CLI:
```bash
go build -o cli
```

3. Move the binary to your PATH:
```bash
mv cli /usr/local/bin/
```

## Getting Started

### Authentication

To authenticate with Major:

```bash
cli login
```

This will open your browser to complete the authentication flow. Once authenticated, your credentials are securely stored in your system keychain.

### Check authentication status

To verify you're logged in and see your user information:

```bash
cli whoami
```

### Logout

To revoke your CLI token and logout:

```bash
cli logout
```

## Usage

### Available Commands

- `cli login` - Authenticate with Major
- `cli logout` - Revoke your CLI token and logout
- `cli whoami` - Display the current authenticated user
- `cli help` - Get help for any command
- `cli version` - Display the CLI version

### Configuration

By default, the CLI uses a configuration file located at `configs/local.json`. You can specify a different config file using the `--config` flag:

```bash
cli --config /path/to/config.json <command>
```

## License

[MIT](LICENSE)

