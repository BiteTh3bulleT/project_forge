package gateway

import (
	"fmt"
	"strings"
)

// This file holds deterministic fallback content templates used only when the
// model omitted structured tool calls. Templates are proposed as write intents;
// gateway policy still validates and executes any filesystem operation.

const defaultRecipeMarkdown = `# Skillet Garlic Butter Pasta

## Ingredients
- 8 oz spaghetti
- 3 tbsp butter
- 3 cloves garlic, minced
- 1/4 tsp red pepper flakes
- 1/2 cup pasta water
- 1/3 cup grated parmesan
- Salt and black pepper
- Chopped parsley (optional)

## Steps
1. Boil salted water and cook spaghetti until al dente.
2. Reserve 1/2 cup pasta water, then drain.
3. Melt butter in a skillet, add garlic and pepper flakes, cook 30 seconds.
4. Add spaghetti and a splash of pasta water, toss until glossy.
5. Add parmesan, salt, and pepper; loosen with more pasta water as needed.
6. Serve hot, topped with parsley.`

func pythonBannerScriptTemplate(bannerText, colorA, colorB string) string {
	escaped := strings.ReplaceAll(bannerText, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "`", "'")

	return fmt.Sprintf(`#!/usr/bin/env python3
"""
FORGE scrolling banner.
Attempts "Vegas Lights" first and falls back to system fonts if unavailable.
"""

import math
import tkinter as tk
from tkinter import font as tkfont

TEXT = "%s"
PRIMARY_FONT = "Vegas Lights"
FALLBACK_FONTS = ("Vegas Lights", "Arial Black", "Helvetica", "TkDefaultFont")
FONT_SIZE = 48
SPEED_PX = 3
FRAME_MS = 20
PULSE_MS = 25
PULSE_STEP = 0.09
COLOR_A = "%s"
COLOR_B = "%s"

root = tk.Tk()
root.title("FORGE Banner")
root.configure(bg="black")
root.geometry("1200x220")

canvas = tk.Canvas(root, bg="black", highlightthickness=0)
canvas.pack(fill="both", expand=True)

font_family = PRIMARY_FONT
available = set(tkfont.families())
for candidate in FALLBACK_FONTS:
    if candidate in available:
        font_family = candidate
        break

text_id = canvas.create_text(
    0,
    0,
    text=TEXT,
    fill=COLOR_A,
    font=(font_family, FONT_SIZE, "bold"),
    anchor="w",
)

phase = 0.0


def _hex_to_rgb(v):
    v = v.lstrip("#")
    return int(v[0:2], 16), int(v[2:4], 16), int(v[4:6], 16)


def _blend(c1, c2, t):
    r = int(c1[0] + (c2[0] - c1[0]) * t)
    g = int(c1[1] + (c2[1] - c1[1]) * t)
    b = int(c1[2] + (c2[2] - c1[2]) * t)
    return f"#{r:02x}{g:02x}{b:02x}"


RGB_A = _hex_to_rgb(COLOR_A)
RGB_B = _hex_to_rgb(COLOR_B)


def layout():
    w = canvas.winfo_width()
    h = canvas.winfo_height()
    y = h // 2
    canvas.coords(text_id, w, y)


def tick():
    canvas.move(text_id, -SPEED_PX, 0)
    bbox = canvas.bbox(text_id)
    if bbox and bbox[2] < 0:
        w = canvas.winfo_width()
        y = canvas.winfo_height() // 2
        canvas.coords(text_id, w, y)
    root.after(FRAME_MS, tick)


def pulse():
    global phase
    t = (math.sin(phase) + 1.0) / 2.0
    canvas.itemconfig(text_id, fill=_blend(RGB_A, RGB_B, t))
    phase += PULSE_STEP
    root.after(PULSE_MS, pulse)


root.bind("<Configure>", lambda _evt: layout())
root.after(100, layout)
root.after(FRAME_MS, tick)
root.after(PULSE_MS, pulse)
root.mainloop()
`, escaped, colorA, colorB)
}

func downloadSorterScriptTemplate() string {
	return `#!/usr/bin/env python3
"""Sort files in ~/Downloads into category folders.

Run once:
    python3 ~/Downloads/Python_Scripts/sort_downloads.py --once

Watch continuously:
    python3 ~/Downloads/Python_Scripts/sort_downloads.py --watch
"""

from __future__ import annotations

import argparse
import shutil
import time
from pathlib import Path

DOWNLOADS = Path.home() / "Downloads"
SCRIPT_DIR = DOWNLOADS / "Python_Scripts"

CATEGORIES = {
    "Images": {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".heic"},
    "Documents": {".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".md"},
    "Spreadsheets": {".xls", ".xlsx", ".csv", ".ods"},
    "Presentations": {".ppt", ".pptx", ".odp"},
    "Archives": {".zip", ".tar", ".gz", ".tgz", ".rar", ".7z"},
    "Code": {".py", ".js", ".ts", ".tsx", ".html", ".css", ".json", ".sh", ".go", ".rs"},
    "Media": {".mp3", ".wav", ".flac", ".mp4", ".mov", ".mkv", ".avi"},
    "Installers": {".exe", ".msi", ".dmg", ".pkg", ".deb", ".rpm", ".appimage"},
}


def category_for(path: Path) -> str:
    suffix = path.suffix.lower()
    for category, extensions in CATEGORIES.items():
        if suffix in extensions:
            return category
    return "Misc"


def unique_destination(path: Path) -> Path:
    if not path.exists():
        return path
    stem = path.stem
    suffix = path.suffix
    parent = path.parent
    counter = 2
    while True:
        candidate = parent / f"{stem}_{counter}{suffix}"
        if not candidate.exists():
            return candidate
        counter += 1


def is_ready(path: Path, delay_seconds: float = 0.75) -> bool:
    if not path.is_file():
        return False
    first = path.stat().st_size
    time.sleep(delay_seconds)
    return path.exists() and path.is_file() and path.stat().st_size == first


def sort_once() -> int:
    DOWNLOADS.mkdir(parents=True, exist_ok=True)
    SCRIPT_DIR.mkdir(parents=True, exist_ok=True)
    moved = 0
    for item in sorted(DOWNLOADS.iterdir(), key=lambda p: p.name.lower()):
        if item == SCRIPT_DIR or item.parent == SCRIPT_DIR:
            continue
        if item.is_dir() or item.name.startswith(".") or item.suffix.lower() in {".crdownload", ".part", ".tmp"}:
            continue
        if not is_ready(item):
            continue
        target_dir = DOWNLOADS / category_for(item)
        target_dir.mkdir(parents=True, exist_ok=True)
        target = unique_destination(target_dir / item.name)
        shutil.move(str(item), str(target))
        print(f"Moved {item.name} -> {target.relative_to(DOWNLOADS)}")
        moved += 1
    return moved


def watch(interval_seconds: float) -> None:
    print(f"Watching {DOWNLOADS} every {interval_seconds:.1f}s")
    while True:
        sort_once()
        time.sleep(interval_seconds)


def main() -> None:
    parser = argparse.ArgumentParser(description="Sort files in ~/Downloads into category folders.")
    parser.add_argument("--once", action="store_true", help="Sort once and exit.")
    parser.add_argument("--watch", action="store_true", help="Keep watching and sorting.")
    parser.add_argument("--interval", type=float, default=5.0, help="Watch interval in seconds.")
    args = parser.parse_args()

    if args.watch:
        watch(max(args.interval, 1.0))
    else:
        moved = sort_once()
        print(f"Done. Moved {moved} file(s).")


if __name__ == "__main__":
    main()
`
}

func videoGameJournalHTML() string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Checkpoint Journal</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #08090b;
      --panel: #14171c;
      --panel-2: #1d2229;
      --line: #8b929d;
      --text: #f0f2f5;
      --muted: #aab0ba;
      --gold: #d6b45f;
      --steel: #68717d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        linear-gradient(120deg, rgba(214,180,95,.12), transparent 34%),
        radial-gradient(circle at top right, rgba(139,146,157,.18), transparent 35%),
        var(--bg);
      color: var(--text);
      font-family: "Trebuchet MS", Arial, sans-serif;
    }
    header {
      border-bottom: 1px solid var(--line);
      padding: 28px clamp(18px, 4vw, 56px);
      background: rgba(20,23,28,.9);
    }
    .brand {
      letter-spacing: .22em;
      text-transform: uppercase;
      color: var(--gold);
      font-size: 13px;
      font-weight: 700;
    }
    h1 {
      margin: 10px 0 8px;
      font-size: clamp(36px, 7vw, 86px);
      line-height: .9;
    }
    .deck {
      max-width: 760px;
      color: var(--muted);
      font-size: 18px;
    }
    main {
      display: grid;
      grid-template-columns: minmax(0, 1.6fr) minmax(260px, .7fr);
      gap: 22px;
      padding: 24px clamp(18px, 4vw, 56px) 44px;
    }
    article, aside, .card {
      border: 1px solid var(--line);
      background: linear-gradient(180deg, var(--panel), #0d0f12);
      box-shadow: 0 24px 60px rgba(0,0,0,.35);
    }
    article { padding: clamp(20px, 3vw, 34px); }
    .meta, .tag {
      color: var(--muted);
      font-size: 12px;
      letter-spacing: .12em;
      text-transform: uppercase;
    }
    h2 { margin: 12px 0; font-size: 30px; }
    p { color: #d7dbe1; line-height: 1.65; }
    .hero-shot {
      min-height: 260px;
      margin: 22px 0;
      border: 1px solid var(--steel);
      background:
        linear-gradient(135deg, transparent 0 45%, rgba(214,180,95,.22) 45% 52%, transparent 52%),
        repeating-linear-gradient(90deg, #10141a 0 14px, #171c23 14px 28px);
      display: grid;
      place-items: center;
      color: var(--gold);
      letter-spacing: .18em;
      text-transform: uppercase;
      font-weight: 800;
    }
    aside { padding: 18px; }
    .score {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 12px;
      padding: 14px 0;
      border-bottom: 1px solid rgba(139,146,157,.45);
    }
    .score strong { color: var(--gold); }
    .card { margin-top: 18px; padding: 16px; background: var(--panel-2); }
    footer {
      padding: 18px clamp(18px, 4vw, 56px);
      border-top: 1px solid var(--line);
      color: var(--muted);
    }
    @media (max-width: 800px) {
      main { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <div class="brand">Checkpoint Journal</div>
    <h1>Patch Notes From The Edge</h1>
    <p class="deck">A compact test layout for reviews, boss-run diaries, build notes, and late-night discoveries from the save point.</p>
  </header>
  <main>
    <article>
      <div class="meta">Feature Review / Issue 01</div>
      <h2>The Dungeon Opens With Good Lighting</h2>
      <p>The first room teaches the rule set without a tutorial wall. Enemy silhouettes read clearly, loot is visible without glowing like a billboard, and the combat loop settles into a clean rhythm after the second encounter.</p>
      <div class="hero-shot">Gameplay Capture</div>
      <p>The best detail is restraint. Menus are fast, quest copy is short, and the map marks what matters. It feels like a journal made by someone who actually finished the quest.</p>
    </article>
    <aside>
      <div class="tag">Score Card</div>
      <div class="score"><span>Combat Feel</span><strong>8.7</strong></div>
      <div class="score"><span>World Design</span><strong>9.1</strong></div>
      <div class="score"><span>Inventory Pain</span><strong>Low</strong></div>
      <div class="card">
        <div class="tag">Current Build</div>
        <p>Stable enough for a full run. Watch shader load stutter after fast travel.</p>
      </div>
    </aside>
  </main>
  <footer>Test video-game journal page generated for local FORGE filesystem routing validation.</footer>
</body>
</html>
`
}

func simpleSVGTemplate(subject string) string {
	switch strings.ToLower(strings.TrimSpace(subject)) {
	case "turtle":
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" role="img" aria-label="turtle">
  <rect width="256" height="256" rx="28" fill="#101417"/>
  <ellipse cx="128" cy="136" rx="72" ry="52" fill="#2f7d4a"/>
  <path d="M70 132c14-34 40-52 58-52s44 18 58 52c-21 16-91 16-116 0z" fill="#66a85f"/>
  <path d="M96 102l24 30-26 24M160 102l-24 30 26 24M104 174l24-28 24 28" fill="none" stroke="#1d4f36" stroke-width="8" stroke-linecap="round" stroke-linejoin="round"/>
  <circle cx="206" cy="128" r="24" fill="#4f9a5f"/>
  <circle cx="214" cy="121" r="3.5" fill="#0b1012"/>
  <path d="M58 126c-17-10-28-9-38 2 9 13 22 15 39 7M79 180c-15 6-23 16-21 31 17 0 27-8 32-24M177 180c15 6 23 16 21 31-17 0-27-8-32-24M80 89c-13-12-25-15-39-8 4 16 16 23 34 20" fill="#4f9a5f"/>
  <path d="M73 130c23 17 86 18 111 0" fill="none" stroke="#183d2b" stroke-width="6" stroke-linecap="round"/>
</svg>
`
	case "stitch":
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" role="img" aria-label="stitch">
  <rect width="256" height="256" rx="28" fill="#101417"/>
  <path d="M48 178c24-54 63-83 116-88" fill="none" stroke="#c8d0d8" stroke-width="10" stroke-linecap="round" stroke-dasharray="18 16"/>
  <path d="M63 167c13 10 30 17 51 19 37 3 66-12 88-44" fill="none" stroke="#8b929d" stroke-width="6" stroke-linecap="round" stroke-dasharray="10 13"/>
  <path d="M168 74l44-34 5 55z" fill="#d7dde3" stroke="#f4f6f8" stroke-width="5" stroke-linejoin="round"/>
  <path d="M176 82l-18 31" stroke="#f4f6f8" stroke-width="7" stroke-linecap="round"/>
  <circle cx="59" cy="178" r="8" fill="#d6b45f"/>
  <circle cx="198" cy="140" r="8" fill="#d6b45f"/>
  <path d="M70 198h116" stroke="#333a42" stroke-width="6" stroke-linecap="round"/>
</svg>
`
	default:
		label := strings.ReplaceAll(strings.TrimSpace(subject), `"`, "'")
		if label == "" {
			label = "svg"
		}
		return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" role="img" aria-label="%s">
  <rect width="256" height="256" rx="28" fill="#101417"/>
  <circle cx="128" cy="110" r="48" fill="#8b929d"/>
  <path d="M64 190c28-34 100-34 128 0" fill="none" stroke="#d6b45f" stroke-width="12" stroke-linecap="round"/>
  <text x="128" y="222" text-anchor="middle" font-family="Arial, sans-serif" font-size="22" fill="#f4f6f8">%s</text>
</svg>
`, label, label)
	}
}
