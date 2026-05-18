import test from "node:test";
import assert from "node:assert/strict";

import {
  collectRoutes,
  formatRoutesMarkdown,
  routeAuthPosture,
} from "./generate-api-routes.mjs";

test("collectRoutes emits chi route inventory with auth posture", () => {
  const routes = collectRoutes();

  assertRoute(routes, "GET", "/health", {
    auth: "public",
    conditions: [],
  });
  assertRoute(routes, "GET", "/health/detailed", {
    auth: "api_bearer_when_configured",
    conditions: [],
  });
  assertRoute(routes, "GET", "/metrics", {
    auth: "api_bearer_when_configured",
    conditions: ["EnableMetricsEndpoint"],
  });
  assertRoute(routes, "GET", "/api/meta", {
    auth: "api_bearer_when_configured",
    conditions: [],
  });
  assertRoute(routes, "GET", "/v1/models", {
    auth: "api_bearer_when_configured",
    conditions: ["EnableOpenAICompatAPI"],
  });
  assertRoute(routes, "POST", "/v1/chat/completions", {
    auth: "api_bearer_when_configured",
    conditions: ["EnableOpenAICompatAPI"],
  });
});

test("formatRoutesMarkdown records source command and auth legend", () => {
  const markdown = formatRoutesMarkdown([
    { method: "GET", path: "/health", auth: "public", conditions: [] },
    {
      method: "GET",
      path: "/health/detailed",
      auth: "api_bearer_when_configured",
      conditions: [],
    },
  ]);

  assert.match(markdown, /Generated from `services\/core\/internal\/api\/routes\.go`/);
  assert.match(markdown, /Regenerate with `node scripts\/generate-api-routes\.mjs`/);
  assert.match(markdown, /\| GET \| `\/health` \| Public \| Always mounted \|/);
  assert.match(markdown, /\| GET \| `\/health\/detailed` \| Bearer token when `APIToken` is configured/);
});

function assertRoute(routes, method, path, expected) {
  const route = routes.find((candidate) => candidate.method === method && candidate.path === path);
  assert.ok(route, `missing ${method} ${path}`);
  assert.equal(route.auth, expected.auth, `${method} ${path} auth`);
  assert.deepEqual(route.conditions, expected.conditions, `${method} ${path} conditions`);
  assert.equal(routeAuthPosture(route), routeAuthPosture(expected));
}
