#!/bin/bash
# PreToolUse hook that auto-approves read-only MCP tools.
# Reads tool_name from stdin JSON, checks via CLI.

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')

if [ -z "$TOOL_NAME" ]; then
  exit 0
fi

# Strip the MCP server prefix to get the actual tool name
# e.g., "mcp__plugin_major_major-resources__postgresql_psql" -> "postgresql_psql"
ACTUAL_TOOL=$(echo "$TOOL_NAME" | sed 's/^mcp__plugin_major_major-resources__//')

major mcp check-readonly "$ACTUAL_TOOL" 2>/dev/null
