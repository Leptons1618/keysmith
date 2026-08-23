#!/usr/bin/env node
// Launcher for the keysmith npm package. Downloads the platform binary
// from GitHub Releases on first use, caches it, and execs it.
"use strict";

const { spawnSync } = require("child_process");
const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");

const REPO = "Leptons1618/keysmith";
const { version } = require(path.join(__dirname, "..", "package.json"));
const TAG = `v${version}`;

function platformTriple() {
  const plat = { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform];
  const arch = { x64: "amd64", arm64: "arm64" }[process.arch];
  if (!plat || !arch) {
    fail(`No prebuilt binary for ${process.platform}/${process.arch}.`);
  }
  return { plat, arch };
}

function wantsTUI() {
  if (process.env.KEYSMITH_TUI === "1") return true;
  if (path.basename(process.argv[1] || "") === "keysmith-tui") return true;
  // The binary understands --tui/--gui itself; peek so we fetch the right one.
  return process.argv.slice(2).includes("--tui");
}

function targetPath(plat, arch, tui) {
  const ext = plat === "windows" ? ".exe" : "";
  const kind = tui ? "-tui" : "";
  const name = `keysmith${kind}-${TAG}-${plat}-${arch}${ext}`;
  const cache = path.join(os.homedir(), ".cache", "keysmith", TAG);
  return { cache, file: path.join(cache, name), name };
}

function download(url, redirects) {
  return new Promise((resolve, reject) => {
    if ((redirects || 0) > 5) return reject(new Error("too many redirects"));
    https.get(url, { headers: { "user-agent": `${REPO} npm launcher` } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        return resolve(download(res.headers.location, (redirects || 0) + 1));
      }
      if (res.statusCode !== 200) {
        res.resume();
        return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
      }
      const chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    }).on("error", reject);
  });
}

function fail(msg) {
  console.error(`keysmith: ${msg}`);
  console.error(`Binaries: https://github.com/${REPO}/releases`);
  process.exit(1);
}

async function main() {
  const tui = wantsTUI();
  const { plat, arch } = platformTriple();
  const { cache, file, name } = targetPath(plat, arch, tui);
  const url =
    `https://github.com/${REPO}/releases/download/${TAG}/${name}`;

  fs.mkdirSync(cache, { recursive: true });
  if (!fs.existsSync(file)) {
    process.stderr.write(`Fetching ${name} ...\n`);
    let buf;
    try {
      buf = await download(url, 0);
    } catch (err) {
      fail(`download failed: ${err.message}`);
    }
    fs.writeFileSync(file, buf);
    if (plat !== "windows") fs.chmodSync(file, 0o755);
  }

  const done = spawnSync(file, process.argv.slice(2), { stdio: "inherit" });
  if (done.error) fail(`could not run ${file}: ${done.error.message}`);
  process.exit(done.status ?? 1);
}

main();
