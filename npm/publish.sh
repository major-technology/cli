#!/bin/bash
# Publishes the npm packages for a release from GoReleaser's dist/ output.
# Usage: npm/publish.sh <version>   (run from repo root after goreleaser)
# Auth comes from npm trusted publishing (GitHub OIDC), not a token.
set -euo pipefail

VERSION="${1#v}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist/npm"
rm -rf "$OUT"

# Skips versions already on npm so re-running a release is safe.
publish() {
  local name version
  name=$(node -p "require('$1/package.json').name")
  version=$(node -p "require('$1/package.json').version")
  if npm view "$name@$version" version >/dev/null 2>&1; then
    echo "$name@$version already published, skipping"
  else
    npm publish "$1" --access public
  fi
}

# goos/goarch -> node platform/arch
for target in darwin/arm64/arm64 darwin/amd64/x64 linux/arm64/arm64 linux/amd64/x64; do
  IFS=/ read -r goos goarch nodearch <<< "$target"
  name="@major-tech/major-$goos-$nodearch"
  src=$(ls "$ROOT"/dist/major_"$goos"_"$goarch"*/major)
  dir="$OUT/major-$goos-$nodearch"
  mkdir -p "$dir/bin"
  cp "$src" "$dir/bin/major"
  chmod +x "$dir/bin/major"
  cat > "$dir/package.json" <<JSON
{
  "name": "$name",
  "version": "$VERSION",
  "description": "The $goos $nodearch binary for the Major CLI",
  "repository": { "type": "git", "url": "git+https://github.com/major-technology/cli.git" },
  "license": "MIT",
  "os": ["$goos"],
  "cpu": ["$nodearch"]
}
JSON
  publish "$dir"
done

cp -r "$ROOT/npm/major" "$OUT/major"
cp "$ROOT/README.md" "$ROOT/LICENSE" "$OUT/major/"
node -e '
  const fs = require("fs");
  const [file, version] = process.argv.slice(1);
  const pkg = JSON.parse(fs.readFileSync(file));
  pkg.version = version;
  for (const dep in pkg.optionalDependencies) pkg.optionalDependencies[dep] = version;
  fs.writeFileSync(file, JSON.stringify(pkg, null, 2) + "\n");
' "$OUT/major/package.json" "$VERSION"
publish "$OUT/major"
