"use strict";

const assert = require("node:assert/strict");
const { Module } = require("node:module");
const path = require("node:path");
const test = require("node:test");

function loadLauncher() {
  const filename = path.join(__dirname, "..", "..", "bin", "keysmith.js");
  const packageFile = path.join(__dirname, "..", "..", "package.json");
  const originalLoad = Module._load;
  Module._load = function (request, parent, isMain) {
    if (parent?.filename === filename && request === packageFile) {
      return { version: "1.2.3" };
    }
    return originalLoad.call(this, request, parent, isMain);
  };
  try {
    return require(filename);
  } finally {
    Module._load = originalLoad;
  }
}

const launcher = loadLauncher();

test("selects only the released GUI assets", () => {
  assert.deepEqual(
    launcher.assetFor("darwin", "arm64", "gui"),
    { platform: "darwin", architecture: "arm64", name: `keysmith-v1.2.3-darwin-arm64` },
  );
  assert.deepEqual(
    launcher.assetFor("linux", "x64", "gui"),
    { platform: "linux", architecture: "amd64", name: `keysmith-v1.2.3-linux-amd64` },
  );
  assert.deepEqual(
    launcher.assetFor("win32", "x64", "gui"),
    { platform: "windows", architecture: "amd64", name: `keysmith-v1.2.3-windows-amd64.exe` },
  );
});

test("selects every released TUI asset", () => {
  for (const [nodePlatform, goPlatform] of [
    ["darwin", "darwin"],
    ["linux", "linux"],
    ["win32", "windows"],
  ]) {
    for (const [nodeArch, goArch] of [
      ["x64", "amd64"],
      ["arm64", "arm64"],
    ]) {
      assert.deepEqual(launcher.assetFor(nodePlatform, nodeArch, "tui"), {
        platform: goPlatform,
        architecture: goArch,
        name: `keysmith-tui-v1.2.3-${goPlatform}-${goArch}${goPlatform === "windows" ? ".exe" : ""}`,
      });
    }
  }
});

test("rejects platforms without a released frontend asset", () => {
  assert.throws(
    () => launcher.assetFor("darwin", "x64", "gui"),
    /desktop GUI is not released for darwin\/x64/,
  );
  assert.throws(
    () => launcher.assetFor("freebsd", "arm64", "tui"),
    /terminal UI is not released for freebsd\/arm64/,
  );
});

test("parses explicit boolean frontend flags", () => {
  assert.deepEqual(launcher.parseLauncherArgs(["--tui=true"]), {
    mode: "tui",
    args: ["--tui=true"],
    versionOnly: false,
  });
  assert.deepEqual(launcher.parseLauncherArgs(["--tui=false", "--gui=true"]), {
    mode: "gui",
    args: ["--tui=false", "--gui=true"],
    versionOnly: false,
  });
  assert.throws(() => launcher.parseLauncherArgs(["--tui=maybe"]), /--tui=maybe/);
});

test("injects the TUI flag for the keysmith-tui command", () => {
  assert.deepEqual(
    launcher.parseLauncherArgs([], { invocation: "/usr/bin/keysmith-tui" }),
    { mode: "tui", args: ["--tui"], versionOnly: false },
  );
  assert.deepEqual(
    launcher.parseLauncherArgs(["--tui=false"], { invocation: "/usr/bin/keysmith-tui" }),
    { mode: "gui", args: ["--tui=false"], versionOnly: false },
  );
});

test("rejects conflicting frontend flags", () => {
  assert.throws(
    () => launcher.parseLauncherArgs(["--gui", "--tui=true"]),
    /--gui and --tui cannot be used together/,
  );
});

test("recognizes version without downloading", () => {
  assert.deepEqual(launcher.parseLauncherArgs(["--version"]), {
    mode: "gui",
    args: ["--version"],
    versionOnly: true,
  });
  assert.deepEqual(launcher.parseLauncherArgs(["--version=false"]), {
    mode: "gui",
    args: ["--version=false"],
    versionOnly: false,
  });
});

test("prints the package version without creating a binary cache", () => {
  const fs = require("node:fs");
  const os = require("node:os");
  const { spawnSync } = require("node:child_process");
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "keysmith-package-"));
  const packageDirectory = path.join(directory, "package");
  const home = path.join(directory, "home");
  fs.mkdirSync(path.join(packageDirectory, "bin"), { recursive: true });
  fs.mkdirSync(home);
  fs.copyFileSync(
    path.join(__dirname, "..", "..", "bin", "keysmith.js"),
    path.join(packageDirectory, "bin", "keysmith.js"),
  );
  fs.writeFileSync(
    path.join(packageDirectory, "package.json"),
    JSON.stringify({ version: "9.8.7" }),
  );

  try {
    const result = spawnSync(
      process.execPath,
      [path.join(packageDirectory, "bin", "keysmith.js"), "--version"],
      {
        env: { ...process.env, HOME: home, USERPROFILE: home },
        encoding: "utf8",
      },
    );
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout, "keysmith 9.8.7\n");
    assert.equal(result.stderr, "");
    assert.equal(fs.existsSync(path.join(home, ".cache")), false);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test("writes a downloaded binary atomically into the cache", () => {
  const fs = require("node:fs");
  const os = require("node:os");
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "keysmith-launcher-"));
  const file = path.join(directory, "keysmith");
  try {
    launcher.writeBinaryAtomically(file, Buffer.from("binary"), 0o755);
    assert.equal(fs.readFileSync(file, "utf8"), "binary");
    assert.equal(fs.statSync(file).mode & 0o777, 0o755);
    assert.deepEqual(fs.readdirSync(directory), ["keysmith"]);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});