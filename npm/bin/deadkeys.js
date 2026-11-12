#!/usr/bin/env node
// Downloads deadkeys.exe from the GitHub release that matches this
// package's version, checks it against SHA256SUMS.txt, and starts it.
// The exe and config.json live in %LOCALAPPDATA%\DeadKeys, so rules
// survive npm clearing its cache.
"use strict";

const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { spawn } = require("child_process");

const REPO = "Visalan-H/Keyboard-issue-lenovo";
const version = "v" + require("../package.json").version;
// DEADKEYS_RELEASE_URL points the download somewhere else, for testing.
const base = process.env.DEADKEYS_RELEASE_URL || `https://github.com/${REPO}/releases/download/${version}`;

function fail(msg) {
  console.error("deadkeys: " + msg);
  process.exit(1);
}

// get fetches url into a Buffer, following redirects.
function get(url, redirects = 5) {
  const lib = url.startsWith("http:") ? require("http") : require("https");
  return new Promise((resolve, reject) => {
    lib
      .get(url, { headers: { "User-Agent": "deadkeys-npm" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirects === 0) return reject(new Error("too many redirects"));
          return resolve(get(new URL(res.headers.location, url).toString(), redirects - 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`${url} returned HTTP ${res.statusCode}`));
        }
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject)
      .setTimeout(60000, function () {
        this.destroy(new Error(`${url} stopped responding`));
      });
  });
}

async function download(exe) {
  console.log(`Downloading DeadKeys ${version}...`);
  const [sums, bin] = await Promise.all([get(`${base}/SHA256SUMS.txt`), get(`${base}/deadkeys.exe`)]);
  // Lines look like "<hash>  deadkeys.exe", or "<hash> *deadkeys.exe"
  // when written in binary mode.
  const entry = sums
    .toString("utf8")
    .split("\n")
    .map((l) => l.trim().match(/^([0-9a-fA-F]{64})\s+\*?(.+)$/))
    .find((m) => m && m[2] === "deadkeys.exe");
  if (!entry) throw new Error("SHA256SUMS.txt has no entry for deadkeys.exe");
  const want = entry[1].toLowerCase();
  const got = crypto.createHash("sha256").update(bin).digest("hex");
  if (got !== want) throw new Error(`checksum mismatch for deadkeys.exe (expected ${want}, got ${got})`);

  const tmp = exe + ".download";
  fs.writeFileSync(tmp, bin);
  fs.renameSync(tmp, exe);
}

// removeOldVersions deletes folders left by earlier versions. One whose
// exe is still running can't be deleted; it is skipped and tried next time.
function removeOldVersions(dir, keep) {
  for (const name of fs.readdirSync(dir)) {
    if (/^v\d/.test(name) && name !== keep) {
      try {
        fs.rmSync(path.join(dir, name), { recursive: true });
      } catch {}
    }
  }
}

async function main() {
  if (process.platform !== "win32") fail("DeadKeys only runs on Windows.");

  const dir = path.join(process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local"), "DeadKeys");
  // Each version gets its own folder so the process keeps the name
  // deadkeys.exe in Task Manager.
  const exe = path.join(dir, version, "deadkeys.exe");
  fs.mkdirSync(path.dirname(exe), { recursive: true });

  if (!fs.existsSync(exe)) {
    try {
      await download(exe);
    } catch (err) {
      fail(`${err.message}\nDownload it yourself from https://github.com/${REPO}/releases`);
    }
    removeOldVersions(dir, version);
  }

  // A --config given by the user comes later and wins.
  const args = ["--config", path.join(dir, "config.json"), ...process.argv.slice(2)];
  const child = spawn(exe, args, { detached: true, stdio: "ignore" });
  child.on("error", (err) => fail(`could not start ${exe}: ${err.message}`));
  child.on("spawn", () => {
    child.unref();
    console.log("DeadKeys is running. Look for the blue keycap in the tray.");
    console.log(`Files: ${dir}`);
  });
}

main();
