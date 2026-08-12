const { MarkdownView, Plugin, normalizePath } = require("obsidian");

const LIMITS = Object.freeze({
  selection: 8192,
  line: 2048,
  openNotes: 64,
  links: 256,
  embeds: 128,
  tags: 256,
  headings: 256,
  textField: 512
});
const HEARTBEAT_MS = 30000;

function boundedText(value, limit) {
  const text = typeof value === "string" ? value : "";
  return text.length <= limit ? text : text.slice(0, limit);
}

function boundedStrings(values, limit, textLimit = LIMITS.textField) {
  if (!Array.isArray(values)) return [];
  return [
    ...new Set(
      values
        .filter((value) => typeof value === "string" && value)
        .map((value) => boundedText(value, textLimit))
    )
  ].slice(0, limit);
}

function boundedRefs(values, limit) {
  if (!Array.isArray(values)) return [];
  return values.slice(0, limit).map((value) => ({
    link: boundedText(value?.link, LIMITS.textField),
    displayText: boundedText(value?.displayText, LIMITS.textField)
  }));
}

function boundedHeadings(values) {
  if (!Array.isArray(values)) return [];
  return values.slice(0, LIMITS.headings).map((value) => ({
    heading: boundedText(value?.heading, LIMITS.textField),
    level: Number.isInteger(value?.level) ? value.level : null
  }));
}

function boundedCursor(value) {
  if (!value || typeof value !== "object") return null;
  const line = Number.isSafeInteger(value.line) && value.line >= 0
    ? value.line
    : 0;
  const ch = Number.isSafeInteger(value.ch) && value.ch >= 0
    ? value.ch
    : 0;
  return { line, ch };
}

module.exports = class ForgeEngineMcpPlugin extends Plugin {
  async onload() {
    this.pending = null;
    this.lastSignature = "";
    this.lastWrittenAt = 0;
    this.status = this.addStatusBarItem();
    this.status.addClass("forge-engine-mcp-status");
    this.setStatus("initializing");

    const update = () => this.scheduleContextWrite();
    this.registerEvent(this.app.workspace.on("active-leaf-change", update));
    this.registerEvent(this.app.workspace.on("file-open", update));
    this.registerEvent(this.app.workspace.on("editor-change", update));
    this.registerEvent(this.app.metadataCache.on("changed", update));
    this.registerInterval(window.setInterval(update, 5000));

    await this.writeContext();
  }

  onunload() {
    if (this.pending) window.clearTimeout(this.pending);
  }

  setStatus(state, detail = "") {
    if (!this.status) return;
    this.status.setText(state === "ready" ? "FORGE context" : "FORGE context !");
    this.status.setAttribute(
      "aria-label",
      detail || `FORGE editor context: ${state}`
    );
    this.status.toggleClass("is-error", state === "error");
  }

  scheduleContextWrite() {
    if (this.pending) window.clearTimeout(this.pending);
    this.pending = window.setTimeout(() => {
      this.pending = null;
      void this.writeContext();
    }, 200);
  }

  buildContext() {
    const view = this.app.workspace.getActiveViewOfType(MarkdownView);
    const file = this.app.workspace.getActiveFile();
    const editor = view?.editor;
    const cache = file ? this.app.metadataCache.getFileCache(file) : null;
    const leaves = this.app.workspace.getLeavesOfType("markdown");
    const selection = editor?.getSelection() ?? "";
    const cursor = boundedCursor(editor?.getCursor());
    const openNotes = leaves
      .map((leaf) => leaf.view?.file?.path)
      .filter(Boolean);

    return {
      protocol: 2,
      schema: "forge.obsidian.editor-context",
      authority: {
        classification: "non_canonical_evidence",
        mayMutateCanonicalState: false,
        requiresGovernedAdmission: true
      },
      provenance: {
        source: "obsidian",
        pluginId: boundedText(this.manifest.id, LIMITS.textField),
        pluginVersion: boundedText(this.manifest.version, LIMITS.textField),
        vault: boundedText(this.app.vault.getName(), LIMITS.textField)
      },
      activeNote: file?.path
        ? boundedText(file.path, LIMITS.textField)
        : null,
      selection: boundedText(selection, LIMITS.selection),
      selectionTruncated: selection.length > LIMITS.selection,
      cursor,
      line: editor && cursor
        ? boundedText(editor.getLine(cursor.line), LIMITS.line)
        : null,
      openNotes: boundedStrings(openNotes, LIMITS.openNotes),
      frontmatterKeys: cache?.frontmatter
        ? boundedStrings(Object.keys(cache.frontmatter).sort(), 128)
        : [],
      links: boundedRefs(cache?.links, LIMITS.links),
      embeds: boundedRefs(cache?.embeds, LIMITS.embeds),
      tags: boundedStrings(
        (cache?.tags ?? []).map((value) => value?.tag).filter(Boolean),
        LIMITS.tags
      ),
      headings: boundedHeadings(cache?.headings),
      limits: LIMITS
    };
  }

  async writeContext() {
    try {
      const payload = this.buildContext();
      const signature = JSON.stringify(payload);
      const now = Date.now();
      if (
        signature === this.lastSignature &&
        now - this.lastWrittenAt < HEARTBEAT_MS
      ) {
        return;
      }

      const context = {
        ...payload,
        generatedAt: new Date(now).toISOString()
      };
      const contextPath = normalizePath(
        `${this.manifest.dir}/context.json`
      );
      await this.app.vault.adapter.write(
        contextPath,
        `${JSON.stringify(context, null, 2)}\n`
      );
      this.lastSignature = signature;
      this.lastWrittenAt = now;
      this.setStatus("ready");
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error("[forge-engine-mcp] context write failed", error);
      this.setStatus("error", `FORGE editor context write failed: ${message}`);
    }
  }
};
