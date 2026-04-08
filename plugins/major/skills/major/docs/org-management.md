# Organization Management

Organizations group users, apps, and resources. Each user can belong to multiple organizations.

## List Organizations

```bash
major org list
```

Lists all organizations the user belongs to. The default org is marked.

```bash
major org list --json
```

Returns JSON with org IDs for programmatic use:
```json
[{"id":"uuid","name":"My Org","isSelected":true}]
```

## Show Current Organization

```bash
major org whoami
```

Displays the currently selected default organization.

## Select Default Organization

### Non-interactive (recommended for AI)

```bash
major org select --id "organization-uuid"
```

Selects the default organization by ID. Use `major org list --json` to get the ID.

### Interactive

```bash
major org select
```

Opens a TUI picker for selecting the default organization. The user must run this in their terminal.

The default organization determines which org's apps and resources are shown when running other commands. The selection is stored in the system keychain.
