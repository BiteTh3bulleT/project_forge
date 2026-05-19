#!/usr/bin/env node

const allowMissing = process.argv.includes("--allow-missing");

const required = [
  {
    name: "FORGE_POSTGRES_TEST_DSN",
    description: "local-optional / CI-required Postgres integration tests",
  },
  {
    name: "FORGE_QDRANT_TEST_URL",
    description: "local-optional / CI-required Qdrant shadow-vector integration tests",
  },
  {
    name: "FORGE_REDIS_TEST_ADDR",
    description: "local-optional / CI-required Redis ephemeral-boundary integration tests",
  },
];

const missing = required.filter((entry) => !process.env[entry.name]);

for (const entry of required) {
  const state = process.env[entry.name] ? "set" : "missing";
  console.log(`${entry.name}: ${state} (${entry.description})`);
}

function printLocalIntegrationHelp() {
  console.log("");
  console.log("Local optional integration test setup:");
  console.log("  docker compose up -d postgres qdrant redis");
  console.log("");
  console.log("PowerShell:");
  console.log("  $env:FORGE_POSTGRES_TEST_DSN='postgres://forge:forge_dev_password@127.0.0.1:5432/forge?sslmode=disable'");
  console.log("  $env:FORGE_QDRANT_TEST_URL='http://127.0.0.1:6333'");
  console.log("  $env:FORGE_REDIS_TEST_ADDR='127.0.0.1:6379'");
  console.log("  npm run test:integration");
  console.log("");
  console.log("Bash:");
  console.log("  export FORGE_POSTGRES_TEST_DSN='postgres://forge:forge_dev_password@127.0.0.1:5432/forge?sslmode=disable'");
  console.log("  export FORGE_QDRANT_TEST_URL='http://127.0.0.1:6333'");
  console.log("  export FORGE_REDIS_TEST_ADDR='127.0.0.1:6379'");
  console.log("  npm run test:integration");
}

if (missing.length > 0 && !allowMissing) {
  console.error(
    `Missing integration environment variables: ${missing
      .map((entry) => entry.name)
      .join(", ")}`,
  );
  printLocalIntegrationHelp();
  process.exit(1);
}

if (missing.length > 0) {
  console.log("Integration services are not required for default validation.");
  printLocalIntegrationHelp();
}
