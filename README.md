# Major CLI

The official command-line interface for [Major](https://major.build) — the platform for deploying and managing secure internal tools.

Major empowers engineering teams to:
- **Deploy instantly**: Ship code to hosted infrastructure with a single command.
- **Secure access**: Manage RBAC and permissions for all your internal apps.
- **Connect resources**: Securely route traffic to internal databases and APIs.

For comprehensive guides, command references, and API documentation, visit our **[Official Documentation](https://docs.major.build/)**.

## Installation

### Direct Install

```bash
curl -fsSL https://install.major.build | bash
```

### Homebrew

```bash
brew tap major-technology/tap
brew install major-technology/tap/major
```

### Updating

Update to the latest version automatically, regardless of install method:

```bash
major update
```

## Quick Start

**1. Authenticate**
Log in to your Major account. This stores your credentials securely in your system keychain.

```bash
major user login
```

The CLI resolves credentials in this order:

1. `MAJOR_TOKEN`, if set to a non-empty value
2. The system keychain entry created by `major user login`

`MAJOR_API_URL` overrides the embedded API base URL and should include the `/cli` path (for example `http://localhost:3001/cli`). A trailing slash is ignored. Other embedded defaults are unchanged when this variable is unset.

When `MAJOR_TOKEN` is set, `major user login`, `major user logout`, and `major user token` refuse to run because the credential is externally managed. Ordinary commands continue to send `Authorization: Bearer <token>` using the injected value.

**2. Create a new App**
Scaffolds a new Major application in your current directory. You'll be prompted to choose a template.

```bash
major app create
```

**3. Start Development**
Installs dependencies (`pnpm install`) and starts the local development server (`pnpm dev`).

```bash
major app start
```

## Documentation

For detailed usage instructions, configuration options, and full command references, please visit the [Major Documentation](https://docs.major.build/).

## License

[MIT](LICENSE)
