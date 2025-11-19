# What is Major?

Major is a platform that lets you easily deploy and manage access to applications you build locally. Major is designed for engineers building internal tools to quickly:

1. Provision hosted compute for their apps
2. Manage access and permissions to those apps
3. Connect apps securely, with RBAC, to internal resources (db's, api's, etc.)

## Installation

### Homebrew
```bash
brew tap major-technology/tap
brew install major-technology/tap/major
```

### Direct Install
```bash
curl -fsSL https://raw.githubusercontent.com/major-technology/cli/main/install.sh | bash
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

## Development Workflow

### Cloud & Deployment
*   **`major app deploy`**
    Commits changes, pushes to your repository, and triggers a deployment to the Major cloud.

*   **`major app info`**
    Displays the ID of the current application.

*   **`major app clone`**
    Interactively select an existing application from your organization to clone locally.

*   **`major app editor`**
    Opens the visual application editor in your browser for the current app.

### Resources & Environment
*   **`major resource create`**
    Opens the Major web console to provision new cloud resources (Postgres, Redis, etc.).

*   **`major resource manage`**
    Interactively select which provisioned resources should be connected to your current app. Updates your project configuration.

*   **`major app generate_env`**
    Pulls environment variables from your connected resources and generates a local `.env` file.

## Organization & Config

*   **`major org list`**
    List all organizations you belong to and see which one is currently active.

*   **`major org select`**
    Interactively switch your active organization context.

*   **`major git config`**
    Configure your GitHub username for git integrations.

## License

[MIT](LICENSE)
