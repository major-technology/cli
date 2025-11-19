# What is Major?

Major is a platform that let's you easily deploy and manage access to applications you build locally. Major is designed for engineers building internal tools to quickly:
1. Provision hosted compute for their apps
2. Manage access to those apps
3. Connect apps securely, with RBAC, to internal resources (db's, api's, etc.)

## Major CLI

`major` is Major on the command line. It brings authentication, app creation, and app deployment directly to your own development environment so that you can create internal apps with ease.

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
go build -o major
```

3. Move the binary to your PATH:
```bash
mv major /usr/local/bin/
```

## Getting Started

### Authentication

To authenticate with Major:

```bash
major user login
```

This will open your browser to complete the authentication flow. Once authenticated, your credentials are securely stored in your system keychain.

### Check authentication status

To verify you're logged in and see your user information:

```bash
major user whoami
```

### Logout

To revoke your CLI token and logout:

```bash
major user logout
```

### Creating and Working on Applications



## License

[MIT](LICENSE)

