# Organization Management

Organizations group users, apps, and resources. Each user can belong to multiple organizations.

## List Organizations

```bash
major org list
```

Lists all organizations the user belongs to. The default org is marked.

## Show Current Organization

```bash
major org whoami
```

Displays the currently selected default organization.

## Select Default Organization

```bash
major org select
```

This is **interactive** — opens a TUI picker for selecting the default organization. The selection is stored in the system keychain.

The default organization determines which org's apps and resources are shown when running other commands.
