#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const path = require("node:path");

const platformMap = {
  darwin: "darwin",
  linux: "linux",
};

const archMap = {
  arm64: "arm64",
  x64: "amd64",
};

const platform = platformMap[process.platform];
const arch = archMap[process.arch];

if (!platform || !arch) {
  console.error(`gitid does not support ${process.platform}/${process.arch} yet.`);
  process.exit(1);
}

const binary = path.join(__dirname, "..", "dist", `gitid-${platform}-${arch}`);
const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error(`Unable to run gitid binary at ${binary}: ${result.error.message}`);
  process.exit(1);
}

if (typeof result.status === "number") {
  process.exit(result.status);
}

process.exit(1);
