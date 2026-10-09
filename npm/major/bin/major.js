#!/usr/bin/env node
const { spawnSync } = require("child_process");

const pkg = `@major-tech/major-${process.platform}-${process.arch}`;
let binary;
try {
  binary = require.resolve(`${pkg}/bin/major`);
} catch {
  console.error(`major: no prebuilt binary for ${process.platform}-${process.arch} (missing ${pkg}).`);
  console.error("If you installed with --no-optional or --omit=optional, reinstall without it.");
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) throw result.error;
if (result.signal) process.kill(process.pid, result.signal);
process.exit(result.status);
