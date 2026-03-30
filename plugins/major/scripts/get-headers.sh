#!/bin/bash
# headersHelper script for Major CLI plugin.
# Outputs JSON headers for MCP server authentication.
# Called by Claude Code at MCP connection time.

TOKEN=$(major user token 2>/dev/null)
ORG=$(major org id 2>/dev/null)

if [ -z "$TOKEN" ] || [ -z "$ORG" ]; then
  echo '{"x-major-error": "Not authenticated. Run: major user login"}' >&2
  exit 1
fi

echo "{\"Authorization\": \"Bearer $TOKEN\", \"x-major-org-id\": \"$ORG\"}"
