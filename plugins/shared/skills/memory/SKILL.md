---
name: using-memory
description: Use this skill when you need to recall knowledge about the organization's resources, data patterns, or conventions from previous sessions. Memory is read-only — you can view and search existing memories but cannot create, edit, or delete them.
---

# Memory Tools

Memory tools let you access knowledge recorded in previous coding sessions. Facts stored here are available to all sessions in this organization.

**Memory is read-only for this agent.** You can view and search existing memory files to inform your work, but you cannot create, edit, or delete them.

## Tools (on major-platform MCP server)

| Tool                               | Purpose                          |
| ---------------------------------- | -------------------------------- |
| `mcp__major-platform__memory_view` | List files or view file contents |
| `mcp__major-platform__memory_grep` | Search memory files by content   |

## File Structure

```
memory/
  organization/              # Org-wide knowledge
    api-patterns.md          # Common API patterns, conventions
    data-model.md            # Key data model facts
    ...
  resources/<resourceId>/    # Resource-specific knowledge
    schema-notes.md          # Schema details, column meanings
    query-patterns.md        # Common query patterns
    ...
```

## When to Use

Check memory at the start of a session or when working with resources to see if previous sessions recorded useful context (schema details, API patterns, conventions).

## Examples

**View all memory files:**

```
memory_view(path: "memory/")
```

**View resource-specific files:**

```
memory_view(path: "memory/resources/abc-123-def/")
```

**Search for relevant knowledge:**

```
memory_grep(pattern: "users table", file_glob: "memory/organization/**/*.md")
```
