#!/usr/bin/env bash
# Find major CLI - check common install locations
MAJOR=""
for p in "$HOME/.major/bin/major" "$HOME/go/bin/major" /usr/local/bin/major; do
  [ -x "$p" ] && MAJOR="$p" && break
done
[ -z "$MAJOR" ] && MAJOR=$(PATH="$HOME/.major/bin:$HOME/go/bin:/usr/local/bin:$PATH" command -v major 2>/dev/null)
[ -z "$MAJOR" ] && exit 1

TOKEN=$("$MAJOR" user ensure-auth)
[ -z "$TOKEN" ] && exit 1
ORG=$("$MAJOR" org id 2>/dev/null)
[ -z "$ORG" ] && exit 1
echo "{\"Authorization\": \"Bearer $TOKEN\", \"x-major-org-id\": \"$ORG\"}"
