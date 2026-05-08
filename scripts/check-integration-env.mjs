#!/usr/bin/env node

const allowMissing = process.argv.includes("--allow-missing");

const required = [
  {
    name: "FORGE_POSTGRES_TEST_DSN",
    description: "optional Postgres integration tests",
  },
  {
    name: "FORGE_QDRANT_TEST_URL",
    description: "optional Qdrant shadow-vector integration tests",
  },
  {
    name: "FORGE_REDIS_TEST_ADDR",
    description: "optional Redis ephemeral-boundary integration tests",
  },
];

const missing = required.filter((entry) => !process.env[entry.name]);

for (const entry of required) {
  const state = process.env[entry.name] ? "set" : "missing";
  console.log(`${entry.name}: ${state} (${entry.description})`);
}

if (missing.length > 0 && !allowMissing) {
  console.error(
    `Missing integration environment variables: ${missing
      .map((entry) => entry.name)
      .join(", ")}`,
  );
  process.exit(1);
}

if (missing.length > 0) {
  console.log("Integration services are not required for default validation.");
}
