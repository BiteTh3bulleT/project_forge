#!/usr/bin/env node
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const routesSource = "services/core/internal/api/routes.go";
const metricsSource = "services/core/internal/api/metrics.go";
const inventoryTestSource = "services/core/internal/api/server_route_inventory_test.go";
const outputPath = "docs/api/routes.md";

const methodNames = new Map([
  ["Get", "GET"],
  ["Post", "POST"],
  ["Patch", "PATCH"],
  ["Put", "PUT"],
  ["Delete", "DELETE"],
]);

export function collectRoutes() {
  const functions = collectRouteFunctions([
    join(repoRoot, routesSource),
    join(repoRoot, metricsSource),
  ]);
  const routes = [];
  const visited = new Set();

  walkFunction("mountHealthRoutes", rootContext(), functions, routes, visited);
  walkFunction("mountForgeRoutes", rootContext(), functions, routes, visited);
  walkFunction("mountOpenAICompatRoutes", rootContext(), functions, routes, visited);
  walkFunction("mountAPIRoutes", rootContext(), functions, routes, visited);

  const seen = new Set();
  return routes.filter((route) => {
    const key = `${route.method} ${route.path} ${route.conditions.join(",")}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function formatRoutesMarkdown(routes = collectRoutes()) {
  const rows = routes.map((route) => {
    return `| ${route.method} | \`${route.path}\` | ${routeAuthPosture(route)} | ${routeCondition(route)} |`;
  });

  return `# API Routes

Generated from \`${routesSource}\` and \`${metricsSource}\`, with route inventory behavior guarded by \`${inventoryTestSource}\`.

Regenerate with \`node scripts/generate-api-routes.mjs\`.
Check without writing with \`node scripts/generate-api-routes.mjs --check\`.

## Auth Posture

- Public: mounted without \`requireAPIAuth\`.
- Bearer token when \`APIToken\` is configured; transport-open when \`APIToken\` is empty: mounted under \`requireAPIAuth\`. An empty token does not grant semantic authority: only verified loopback peers receive \`local_loopback\` origin proof, while arbitrary remote peers receive no authenticated user/proposer origin and fail closed at FORGE-K authorization.
- Handler-specific checks, approval gates, capability gates, and remote webhook signature/token validation are not expanded here unless visible at the route-mount layer.

## Inventory

| Method | Path | Auth posture | Mount condition |
| --- | --- | --- | --- |
${rows.join("\n")}
`;
}

export function routeAuthPosture(route) {
  if (route.auth === "api_bearer_when_configured") {
    return "Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty";
  }
  return "Public";
}

function routeCondition(route) {
  if (!route.conditions || route.conditions.length === 0) return "Always mounted";
  return route.conditions.map((condition) => `\`${condition}\``).join(", ");
}

function rootContext() {
  return {
    prefix: "",
    auth: "public",
    conditions: [],
  };
}

function collectRouteFunctions(paths) {
  const functions = new Map();
  for (const path of paths) {
    const source = readFileSync(path, "utf8");
    for (const fn of extractServerMethods(source)) {
      functions.set(fn.name, fn.body);
    }
  }
  return functions;
}

function extractServerMethods(source) {
  const lines = source.split(/\r?\n/);
  const methods = [];
  for (let i = 0; i < lines.length; i++) {
    const match = lines[i].match(/^func \(s \*Server\) (mount\w+Routes)\(/);
    if (!match) continue;

    const body = [];
    let depth = braceDelta(lines[i]);
    for (i = i + 1; i < lines.length; i++) {
      depth += braceDelta(lines[i]);
      if (depth <= 0) break;
      body.push(lines[i]);
    }
    methods.push({ name: match[1], body });
  }
  return methods;
}

function walkFunction(name, context, functions, routes, visited) {
  const body = functions.get(name);
  if (!body) return;

  const contextKey = `${name}|${context.prefix}|${context.auth}|${context.conditions.join(",")}`;
  if (visited.has(contextKey)) return;
  visited.add(contextKey);

  const initial = cloneContext(context);
  if (name === "mountMetricsRoutes") {
    initial.conditions = withCondition(initial.conditions, "EnableMetricsEndpoint");
  }

  const stack = [initial];
  for (const rawLine of body) {
    const line = rawLine.trim();
    const current = stack[stack.length - 1];

    const cfgCondition = line.match(/^if s\.cfg\.(\w+) \{/);
    if (cfgCondition) {
      stack.push({
        ...cloneContext(current),
        conditions: withCondition(current.conditions, cfgCondition[1]),
      });
      continue;
    }

    const routeBlock = line.match(/^r\.Route\("([^"]+)",\s*func\(r chi\.Router\) \{/);
    if (routeBlock) {
      stack.push({
        ...cloneContext(current),
        prefix: joinRoute(current.prefix, routeBlock[1]),
      });
      continue;
    }

    if (/^r\.Group\(func\(r chi\.Router\) \{/.test(line)) {
      stack.push(cloneContext(current));
      continue;
    }

    if (/^r\.Use\(s\.requireAPIAuth\)/.test(line)) {
      current.auth = "api_bearer_when_configured";
    }

    const route = line.match(/^r\.(Get|Post|Patch|Put|Delete)\("([^"]+)"/);
    if (route) {
      routes.push({
        method: methodNames.get(route[1]),
        path: joinRoute(current.prefix, route[2]),
        auth: current.auth,
        conditions: [...current.conditions],
      });
    }

    const mountCall = line.match(/^s\.(mount\w+Routes)\(r\)/);
    if (mountCall) {
      walkFunction(mountCall[1], current, functions, routes, visited);
    }

    if (/^\}\)?$/.test(line) || /^\}\)$/.test(line)) {
      if (stack.length > 1) stack.pop();
    }
  }
}

function cloneContext(context) {
  return {
    prefix: context.prefix,
    auth: context.auth,
    conditions: [...context.conditions],
  };
}

function withCondition(conditions, condition) {
  return Array.from(new Set([...conditions, condition]));
}

function joinRoute(prefix, path) {
  const joined = `${prefix.replace(/\/$/, "")}/${path.replace(/^\//, "")}`;
  return joined === "/" ? "/" : joined;
}

function braceDelta(line) {
  let delta = 0;
  for (const char of line) {
    if (char === "{") delta++;
    if (char === "}") delta--;
  }
  return delta;
}

function writeRoutesDoc(markdown) {
  const destination = join(repoRoot, outputPath);
  mkdirSync(dirname(destination), { recursive: true });
  writeFileSync(destination, markdown, "utf8");
}

function checkRoutesDoc(markdown) {
  const destination = join(repoRoot, outputPath);
  if (!existsSync(destination)) {
    console.error(`${outputPath} does not exist. Run node scripts/generate-api-routes.mjs.`);
    process.exit(1);
  }
  const existing = readFileSync(destination, "utf8");
  if (existing !== markdown) {
    console.error(`${outputPath} is out of date. Run node scripts/generate-api-routes.mjs.`);
    process.exit(1);
  }
}

function usage() {
  console.log(`usage: node scripts/generate-api-routes.mjs [--check|--stdout]

Generates ${outputPath} from chi route setup and route inventory test sources.`);
}

if (process.argv[1] && relative(repoRoot, fileURLToPath(import.meta.url)) === relative(repoRoot, process.argv[1])) {
  const markdown = formatRoutesMarkdown(collectRoutes());
  const args = process.argv.slice(2);
  if (args.includes("-h") || args.includes("--help")) {
    usage();
    process.exit(0);
  }
  if (args.length > 1 || args.some((arg) => !["--check", "--stdout"].includes(arg))) {
    usage();
    process.exit(2);
  }
  if (args[0] === "--check") {
    checkRoutesDoc(markdown);
  } else if (args[0] === "--stdout") {
    process.stdout.write(markdown);
  } else {
    writeRoutesDoc(markdown);
    console.log(`wrote ${outputPath}`);
  }
}
