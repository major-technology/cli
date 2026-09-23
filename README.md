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

`--non-interactive` never prompts or opens a browser and does not imply `--yes`. Git remotes set `GIT_TERMINAL_PROMPT=0` and insert OpenSSH `BatchMode=yes` after a single direct `ssh` executable (including a path whose basename is `ssh`). Quoted and escaped arguments are preserved. `GIT_SSH_COMMAND` wrappers (`env ... ssh`), other first executables, and compound commands (`ssh ...; ...`) are rejected; use a direct `ssh` command or run without `--non-interactive`.

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

### Agent and skill commands

`major app list [--editable]`, `major agent list [--editable]`, `major agent get [agent-id]`, `major agent create --name NAME [--description TEXT]`, `major agent run [agent-id] --prompt TEXT [--name TITLE]`, `major agent runs list [--agent ID] [--all-users]`, `major agent runs content RUN-ID [--limit N]`, `major agent runs send RUN-ID --message TEXT`, `major agent runs stop RUN-ID`, `major agent channel connect|pause|resume|delete [agent-id] --type slack`, `major agent permissions resource|app [agent-id] TARGET-ID`, `major skill list [--editable] [--published]`, `major skill get [skill-id]`, and `major skill create` accept `--json` for route-shaped output. In an agent or skill workspace, its `.major/config.json` supplies an omitted matching ID; an explicit ID overrides it. Lists and creates use the organization bound to the API token without a checkout or keyring default. Starting runs, connecting Slack, deleting channels, and creating definitions require a human CLI token; server authorization governs other operations.
