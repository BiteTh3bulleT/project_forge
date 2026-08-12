import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";
import { resolve } from "node:path";

const repoRoot = resolve(import.meta.dirname, "..");
const pluginPath = resolve(
  repoRoot,
  ".obsidian/plugins/forge-engine-mcp/main.js"
);
const pluginSource = readFileSync(pluginPath, "utf8");

function loadPlugin() {
  class MarkdownView {}
  class Plugin {}
  const requiredModules = [];
  const module = { exports: {} };
  const sandbox = {
    console,
    module,
    exports: module.exports,
    require(specifier) {
      requiredModules.push(specifier);
      if (specifier !== "obsidian") {
        throw new Error(`unexpected plugin dependency: ${specifier}`);
      }
      return {
        MarkdownView,
        Plugin,
        normalizePath: (value) => value
      };
    }
  };

  vm.runInNewContext(pluginSource, sandbox, { filename: pluginPath });
  return { PluginClass: module.exports, requiredModules };
}

test("Obsidian context bundle only imports the editor API", () => {
  const { requiredModules } = loadPlugin();
  assert.deepEqual(requiredModules, ["obsidian"]);

  for (const forbiddenSurface of [
    "child_process",
    "node:http",
    "node:https",
    "XMLHttpRequest",
    "WebSocket",
    "fetch("
  ]) {
    assert.equal(
      pluginSource.includes(forbiddenSurface),
      false,
      `plugin must not contain external execution surface ${forbiddenSurface}`
    );
  }
});

test("Obsidian context envelope bounds content and denies authority", () => {
  const { PluginClass } = loadPlugin();
  const longText = "x".repeat(12_000);
  const secretValue = "must-not-enter-context";
  const file = { path: longText };
  const editor = {
    getSelection: () => longText,
    getCursor: () => ({ line: Number.MAX_SAFE_INTEGER + 1, ch: -1 }),
    getLine: () => longText
  };
  const refs = Array.from({ length: 400 }, (_, index) => ({
    link: `${index}:${longText}`,
    displayText: `${index}:${longText}`
  }));
  const headings = Array.from({ length: 400 }, (_, index) => ({
    heading: `${index}:${longText}`,
    level: index % 6 + 1
  }));
  const tags = Array.from({ length: 400 }, (_, index) => ({
    tag: `${index}:${longText}`
  }));
  const frontmatter = Object.fromEntries(
    Array.from({ length: 200 }, (_, index) => [
      `${String(index).padStart(3, "0")}:${longText}`,
      secretValue
    ])
  );
  const openLeaves = Array.from({ length: 100 }, (_, index) => ({
    view: { file: { path: `${index}:${longText}` } }
  }));

  const plugin = new PluginClass();
  plugin.manifest = {
    id: longText,
    version: longText,
    dir: ".obsidian/plugins/forge-engine-mcp"
  };
  plugin.app = {
    workspace: {
      getActiveViewOfType: () => ({ editor }),
      getActiveFile: () => file,
      getLeavesOfType: () => openLeaves
    },
    metadataCache: {
      getFileCache: () => ({
        frontmatter,
        links: refs,
        embeds: refs,
        tags,
        headings
      })
    },
    vault: { getName: () => longText }
  };

  const context = plugin.buildContext();
  assert.equal(context.protocol, 2);
  assert.equal(context.schema, "forge.obsidian.editor-context");
  assert.equal(context.authority.classification, "non_canonical_evidence");
  assert.equal(context.authority.mayMutateCanonicalState, false);
  assert.equal(context.authority.requiresGovernedAdmission, true);
  assert.equal(context.selection.length, 8192);
  assert.equal(context.selectionTruncated, true);
  assert.equal(context.line.length, 2048);
  assert.equal(context.activeNote.length, 512);
  assert.equal(context.provenance.pluginId.length, 512);
  assert.equal(context.provenance.pluginVersion.length, 512);
  assert.equal(context.provenance.vault.length, 512);
  assert.equal(context.cursor.line, 0);
  assert.equal(context.cursor.ch, 0);
  assert.equal(context.openNotes.length, 64);
  assert.equal(context.links.length, 256);
  assert.equal(context.embeds.length, 128);
  assert.equal(context.tags.length, 256);
  assert.equal(context.headings.length, 256);
  assert.equal(context.frontmatterKeys.length, 128);

  for (const value of [
    ...context.openNotes,
    ...context.tags,
    ...context.frontmatterKeys
  ]) {
    assert.ok(value.length <= 512);
  }
  for (const ref of [...context.links, ...context.embeds]) {
    assert.ok(ref.link.length <= 512);
    assert.ok(ref.displayText.length <= 512);
  }
  for (const heading of context.headings) {
    assert.ok(heading.heading.length <= 512);
  }
  assert.equal(JSON.stringify(context).includes(secretValue), false);
});
