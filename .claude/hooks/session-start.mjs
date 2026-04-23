#!/usr/bin/env node

import { execSync } from "node:child_process";

function safeExec(command) {
  try {
    return execSync(command, {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"]
    }).trim();
  } catch {
    return "";
  }
}

const branch = safeExec("git rev-parse --abbrev-ref HEAD");
const status = safeExec("git status --short");
const dirtyFiles = status ? status.split("\n").filter(Boolean) : [];
const previewCount = 8;
const preview = dirtyFiles.slice(0, previewCount).join(", ");
const more = dirtyFiles.length > previewCount ? ` (+${dirtyFiles.length - previewCount} more)` : "";

const contextLines = [
  `FORGE workspace: ${process.env.CLAUDE_PROJECT_DIR ?? process.cwd()}`,
  branch ? `Git branch: ${branch}` : "Git branch: unavailable",
  `Dirty files: ${dirtyFiles.length}${preview ? ` (${preview}${more})` : ""}`,
  "Fast commands: npm run desktop | npm run up | npm run down | npm run build:core | npm run build:desktop | cd services/core && go test ./..."
];

const payload = {
  hookSpecificOutput: {
    hookEventName: "SessionStart",
    additionalContext: contextLines.join("\n")
  }
};

process.stdout.write(JSON.stringify(payload));
