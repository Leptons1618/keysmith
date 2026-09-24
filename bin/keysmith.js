#!/usr/bin/env node
// Launcher for the keysmith npm package. Downloads the platform binary
// from GitHub Releases on first use, caches it, and runs it.
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");

const REPO = "Leptons1618/keysmith";
const { version } = require(path.join(__dirname, "..", "package.json"));
const TAG = `v${version}`;

const GUI_ASSETS = new Set([
  "darwin/arm64",
  "linux/amd64",
  "windows/amd64",
]);
const TUI_ASSETS = new Set([
  "darwin/amd64",
  "darwin/arm64",
  "linux/amd64",
  "linux/arm64",
  "windows/amd64",
  "windows/arm64",
]);

function assetFor(nodePlatform, nodeArch, mode) {
  const platform = { darwin: "darwin", linux: "linux", win32: "windows" }[nodePlatform];
  const architecture = { x64: "amd64", arm64: "arm64" }[nodeArch];
  const frontend = mode === "gui" ? "desktop GUI" : "terminal UI";
  if (!platform || !architecture) {
    throw new Error(`${frontend} is not released for ${nodePlatform}/${nodeArch}`);
  }

  const key = `${platform}/${architecture}`;
  const available = mode === "gui" ? GUI_ASSETS : TUI_ASSETS;
  if (!available.has(key)) {
    throw new Error(`${frontend} is not released for ${nodePlatform}/${nodeArch}`);
  }

  const extension = platform === "windows" ? ".exe" : "";
  const kind = mode === "tui" ? "-tui" : "";
  return {
    platform,
    architecture,
    name: `keysmith${kind}-${TAG}-${platform}-${architecture}${extension}`,
  };
}

function parseBooleanOption(argument, name) {
  if (argument === name) return true;
  const prefix = `${name}=`;
  if (!argument.startsWith(prefix)) return undefined;
  const value = argument.slice(prefix.length);
  if (value !== "true" && value !== "false") {
    throw new Error(`${name}=${value} must be true or false`);
  }
  return value === "true";
}

function isTUIInvocation(invocation) {
  const name = path.basename(invocation || "").toLowerCase();
  return /^keysmith-tui(?:\.cmd|\.exe|\.ps1)?$/.test(name);
}

function parseLauncherArgs(args, options = {}) {
  const invocation = options.invocation ?? process.argv[1] ?? "";
  const env = options.env ?? process.env;
  let tui;
  let gui;
  let versionOnly = false;

  for (const argument of args) {
    if (argument === "--" || !argument.startsWith("-")) break;
    if (argument === "-h" || argument === "--help") break;
    if (argument === "-v" || argument === "--version") {
      versionOnly = true;
      continue;
    }
    const parsedVersion = parseBooleanOption(argument, "--version");
    if (parsedVersion !== undefined) {
      versionOnly = parsedVersion;
      continue;
    }
    const parsedTUI = parseBooleanOption(argument, "--tui");
    if (parsedTUI !== undefined) {
      tui = parsedTUI;
      continue;
    }
    const parsedGUI = parseBooleanOption(argument, "--gui");
    if (parsedGUI !== undefined) {
      gui = parsedGUI;
    }
  }

  if (tui === true && gui === true) {
    throw new Error("--gui and --tui cannot be used together");
  }

  let mode;
  if (tui !== undefined) {
    mode = tui ? "tui" : "gui";
  } else if (gui !== undefined) {
    mode = gui ? "gui" : "tui";
  } else if (env.KEYSMITH_TUI === "1" || isTUIInvocation(invocation)) {
    mode = "tui";
  } else {
    mode = "gui";
  }

  const forwarded = versionOnly ? args : mode === "tui" && tui === undefined ? ["--tui", ...args] : args;
  return { mode, args: forwarded, versionOnly };
}

function cacheTarget(asset) {
  const cache = path.join(os.homedir(), ".cache", "keysmith", TAG);
  return { cache, file: path.join(cache, asset.name) };
}

function writeBinaryAtomically(file, data, mode) {
  const directory = path.dirname(file);
  const temporary = path.join(
    directory,
    `.${path.basename(file)}.${process.pid}.${Math.random().toString(16).slice(2)}.tmp`,
  );
  let descriptor;
  try {
    descriptor = fs.openSync(temporary, "wx", mode);
    fs.writeFileSync(descriptor, data);
    fs.fchmodSync(descriptor, mode);
    fs.fsyncSync(descriptor);
    fs.closeSync(descriptor);
    descriptor = undefined;
    fs.renameSync(temporary, file);
  } catch (error) {
    if (descriptor !== undefined) fs.closeSync(descriptor);
    try {
      fs.rmSync(temporary, { force: true });
    } catch {
      // Preserve the original cache write error.
    }
    throw error;
  }
}

function download(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error("too many redirects"));
    https.get(url, { headers: { "user-agent": `${REPO} npm launcher` } }, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        return resolve(download(response.headers.location, redirects + 1));
      }
      if (response.statusCode !== 200) {
        response.resume();
        return reject(new Error(`HTTP ${response.statusCode} for ${url}`));
      }
      const chunks = [];
      response.on("data", (chunk) => chunks.push(chunk));
      response.on("end", () => resolve(Buffer.concat(chunks)));
      response.on("error", reject);
    }).on("error", reject);
  });
}

function fail(message) {
  console.error(`keysmith: ${message}`);
  console.error(`Binaries: https://github.com/${REPO}/releases`);
  process.exitCode = 1;
}

async function main() {
  let selection;
  try {
    selection = parseLauncherArgs(process.argv.slice(2));
  } catch (error) {
    fail(error.message);
    return;
  }

  if (selection.versionOnly) {
    console.log(`keysmith ${version}`);
    return;
  }

  let asset;
  let target;
  try {
    asset = assetFor(process.platform, process.arch, selection.mode);
    target = cacheTarget(asset);
  } catch (error) {
    fail(error.message);
    return;
  }

  fs.mkdirSync(target.cache, { recursive: true });
  if (!fs.existsSync(target.file)) {
    const url = `https://github.com/${REPO}/releases/download/${TAG}/${asset.name}`;
    process.stderr.write(`Fetching ${asset.name} ...\n`);
    try {
      const contents = await download(url);
      writeBinaryAtomically(target.file, contents, asset.platform === "windows" ? 0o666 : 0o755);
    } catch (error) {
      fail(`download failed: ${error.message}`);
      return;
    }
  }

  const result = spawnSync(target.file, selection.args, { stdio: "inherit" });
  if (result.error) {
    fail(`could not run ${target.file}: ${result.error.message}`);
    return;
  }
  process.exitCode = result.status ?? 1;
}

if (require.main === module) main();

module.exports = {
  TAG,
  assetFor,
  cacheTarget,
  parseLauncherArgs,
  writeBinaryAtomically,
};
