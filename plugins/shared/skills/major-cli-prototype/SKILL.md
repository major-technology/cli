---
name: major-cli-prototype
description: Develop an existing Major app through the Major CLI, locally or in its mounted workspace.
---

Use the shell that executes in the app workspace. Use `--non-interactive` for Major commands and `--json` when consuming results. Never print credentials. Do not use raw API requests to perform app-management steps.

1. Inspect with `major app info --non-interactive --json`.
2. Use `major vars list --non-interactive --json` only when environment values are needed; treat returned values as secrets. Use a named test key for validation, not customer credentials.
3. Edit using normal file tools. Review changes; commit and push explicitly. Never force-push or switch branches implicitly.
4. Deploy only when the user explicitly asks. Use `major app deploy --non-interactive --no-wait --json`; supply `--slug` on first deploy. Record the returned version ID.
5. Use the emitted status command to inspect that deployment. A started deployment is not yet deployed. Use bounded status checks, never an unbounded polling loop.
6. Read bounded logs with `major app logs --non-interactive --json --limit 20` when needed.
7. If inputs or confirmation are missing, provide the named flags only when the user's intent authorizes them. Never treat `--non-interactive` as `--yes`.
8. After an uncertain deploy response, inspect state before retrying. Never assume a timeout means no mutation happened.
