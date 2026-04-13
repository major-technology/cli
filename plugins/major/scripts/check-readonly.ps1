# PreToolUse hook that auto-approves read-only MCP tools.
# Reads tool_name from stdin JSON, checks via CLI.

$InputJson = $input | Out-String
$Parsed = $InputJson | ConvertFrom-Json -ErrorAction SilentlyContinue

if (-not $Parsed -or -not $Parsed.tool_name) { exit 0 }

$ToolName = $Parsed.tool_name

# Strip the MCP server prefix to get the actual tool name
$ActualTool = $ToolName -replace '^mcp__plugin_major_major-resources__', ''

& major mcp check-readonly $ActualTool 2>$null
