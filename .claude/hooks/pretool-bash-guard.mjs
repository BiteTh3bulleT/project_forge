#!/usr/bin/env node

function readStdin() {
  return new Promise((resolve) => {
    let data = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", (chunk) => {
      data += chunk;
    });
    process.stdin.on("end", () => {
      resolve(data);
    });
    process.stdin.resume();
  });
}

const rawInput = await readStdin();
let command = "";

try {
  const parsed = JSON.parse(rawInput || "{}");
  command = parsed?.tool_input?.command ?? "";
} catch {
  command = "";
}

const guardRules = [
  {
    regex: /\bgit\s+reset\s+--hard\b/i,
    reason: "Blocked destructive git reset command."
  },
  {
    regex: /\bgit\s+checkout\s+--\s+/i,
    reason: "Blocked destructive git checkout -- command."
  },
  {
    regex: /\bgit\s+clean\s+-fdx?\b/i,
    reason: "Blocked destructive git clean command."
  },
  {
    regex: /\bsudo\s+rm\s+-rf\s+\/\*?(?:\s|$)/i,
    reason: "Blocked destructive root filesystem delete."
  },
  {
    regex: /(?:^|\s)rm\s+-rf\s+\/\*?(?:\s|$)/i,
    reason: "Blocked destructive root filesystem delete."
  }
];

for (const rule of guardRules) {
  if (rule.regex.test(command)) {
    const payload = {
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "deny",
        permissionDecisionReason: `${rule.reason} Command: ${command.slice(0, 180)}`
      }
    };
    process.stdout.write(JSON.stringify(payload));
    process.exit(0);
  }
}
