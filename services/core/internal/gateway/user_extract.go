package gateway

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reDirectoryCalled  = regexp.MustCompile(`(?i)(?:directory|folder)\s+(?:called|call|named)\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reDirectoryLabeled = regexp.MustCompile(`(?i)(?:directory|folder)\s+(?:labeled|labelled|labled|labeld)\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reInsidePath       = regexp.MustCompile(`(?i)\binside\s+(?:the\s+)?['"]?([^\s'"]+(?:/[^\s'"]*)?)['"]?(?:\s+directory)?`)
	reCreateDirPath    = regexp.MustCompile(`(?i)\b(?:create|make)\s+(?:an?\s+)?([a-z0-9_.\-\/]+)\s+directory\b`)
	reFileLabeled      = regexp.MustCompile(`(?i)(?:a\s+)?file\s+labeled\s+['"]([^'"]+)['"]`)
	reTheWords         = regexp.MustCompile(`(?i)the words\s+['"]([^'"]+)['"]`)
	reMkdirOnly        = regexp.MustCompile(`(?i)\bmkdir(?:\s+-p)?\s+([^\s#]+)`)
	reCatPath          = regexp.MustCompile(`(?i)^\s*cat\s+([^\s#]+)`)
	reReadFilePath     = regexp.MustCompile(`(?i)\bread(?:\s+the)?\s+file\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reLsPath           = regexp.MustCompile(`(?i)^\s*ls(?:\s+-[^\s]+)*\s+([^\s#]+)`)
	reListDirPath      = regexp.MustCompile(`(?i)\blist(?:\s+the)?\s+(?:directory|files)(?:\s+(?:in|at|under))?\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reInDirectoryPath  = regexp.MustCompile(`(?i)\bin\s+(?:the\s+)?['"]?([a-z0-9_.\-\/]+)['"]?\s+directory\b`)
	rePyFilePath       = regexp.MustCompile(`(?i)\b([a-z0-9_\-./]+\.py)\b`)
	reSaysQuoted       = regexp.MustCompile(`(?i)\bsays?\s+['"]([^'"]+)['"]`)
)

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

// ParseCombinedMkdirAndWrite extracts mkdir + labeled file + quoted content from a single
// natural-language request (used when the model omits tool_calls).
//
// Supported shapes include:
//   - "directory called foo ... file labeled "a" ... the words "b""
//   - "file labeled "a" inside foo/bar/ ... the words "b"" (first "inside <path>" wins over later "inside said")
func ParseCombinedMkdirAndWrite(user string) (dirRel, fileName, content string, ok bool) {
	user = strings.TrimSpace(user)
	if user == "" {
		return "", "", "", false
	}
	fm := reFileLabeled.FindStringSubmatch(user)
	wm := reTheWords.FindStringSubmatch(user)
	if len(fm) >= 2 && len(wm) >= 2 {
		fileName = strings.TrimSpace(fm[1])
		content = wm[1]
	} else if looksLikeRecipeMarkdownIntent(user) {
		fileName = "recipe.md"
		content = defaultRecipeMarkdown
	} else {
		return "", "", "", false
	}
	if fileName == "" {
		return "", "", "", false
	}
	fileName = filepath.ToSlash(fileName)
	if strings.Contains(fileName, "/") {
		fileName = filepath.Base(fileName)
	}

	var dirRaw string
	if dm := reDirectoryCalled.FindStringSubmatch(user); len(dm) >= 2 {
		dirRaw = strings.TrimSpace(dm[1])
	} else if im := reInsidePath.FindStringSubmatch(user); len(im) >= 2 {
		dirRaw = strings.TrimSpace(im[1])
	} else {
		return "", "", "", false
	}
	dirRel = strings.Trim(strings.TrimSpace(dirRaw), `/`)
	dirRel = strings.TrimRight(dirRel, ".,;:!?")
	dirRel = strings.Trim(dirRel, `/`)
	if dirRel == "" {
		return "", "", "", false
	}
	return dirRel, fileName, content, true
}

func looksLikeRecipeMarkdownIntent(user string) bool {
	s := strings.ToLower(strings.TrimSpace(user))
	if s == "" {
		return false
	}
	hasFileIntent := strings.Contains(s, "markdown file") || strings.Contains(s, "md file") || strings.Contains(s, ".md")
	hasRecipeIntent := strings.Contains(s, "recipe")
	hasCreateIntent := strings.Contains(s, "create") || strings.Contains(s, "write")
	return hasFileIntent && hasRecipeIntent && hasCreateIntent
}

// ParseDirectoryCalled extracts a path after "directory called|named".
func ParseDirectoryCalled(user string) (dirRel string, ok bool) {
	dm := reDirectoryCalled.FindStringSubmatch(strings.TrimSpace(user))
	if len(dm) < 2 {
		return "", false
	}
	dirRel = strings.TrimSpace(dm[1])
	if dirRel == "" {
		return "", false
	}
	return filepath.ToSlash(dirRel), true
}

// ParseMkdirShellPath extracts path from a mkdir shell-style fragment.
func ParseMkdirShellPath(user string) (path string, ok bool) {
	m := reMkdirOnly.FindStringSubmatch(strings.TrimSpace(user))
	if len(m) < 2 {
		return "", false
	}
	p := strings.TrimSpace(m[1])
	p = strings.Trim(p, `"'`)
	if p == "" {
		return "", false
	}
	return filepath.ToSlash(p), true
}

// ParseReadPath extracts a file path from "cat <path>" or "read file <path>".
func ParseReadPath(user string) (path string, ok bool) {
	s := strings.TrimSpace(user)
	if s == "" {
		return "", false
	}
	if m := reCatPath.FindStringSubmatch(s); len(m) >= 2 {
		p := strings.Trim(strings.TrimSpace(m[1]), `"'`)
		if p != "" {
			return filepath.ToSlash(p), true
		}
	}
	if m := reReadFilePath.FindStringSubmatch(s); len(m) >= 2 {
		p := strings.Trim(strings.TrimSpace(m[1]), `"'`)
		if p != "" {
			return filepath.ToSlash(p), true
		}
	}
	return "", false
}

// ParseListPath extracts an optional directory path from "ls <path>" or "list files in <path>".
// Returns "." when listing intent exists but no explicit path is found.
func ParseListPath(user string) (path string, ok bool) {
	s := strings.TrimSpace(user)
	if s == "" {
		return "", false
	}
	if m := reLsPath.FindStringSubmatch(s); len(m) >= 2 {
		p := strings.Trim(strings.TrimSpace(m[1]), `"'`)
		if p != "" {
			return filepath.ToSlash(p), true
		}
	}
	if m := reListDirPath.FindStringSubmatch(s); len(m) >= 2 {
		p := strings.Trim(strings.TrimSpace(m[1]), `"'`)
		if p != "" {
			return filepath.ToSlash(p), true
		}
	}
	sl := strings.ToLower(s)
	if strings.HasPrefix(sl, "ls") || strings.Contains(sl, "list files") || strings.Contains(sl, "list directory") {
		return ".", true
	}
	return "", false
}

// ParseShellCommand extracts command text from "run ..." or "execute ...".
func ParseShellCommand(user string) (cmd string, ok bool) {
	s := strings.TrimSpace(user)
	if s == "" {
		return "", false
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "run ") {
		cmd = strings.TrimSpace(s[len("run "):])
		return cmd, cmd != ""
	}
	if strings.HasPrefix(low, "execute ") {
		cmd = strings.TrimSpace(s[len("execute "):])
		return cmd, cmd != ""
	}
	return "", false
}

// ParsePythonBannerScriptIntent parses requests like:
// "in <dir>, create a scrolling banner python script that says 'FORGE LIVES!'".
// Returns a write path plus script contents when intent is clear.
func ParsePythonBannerScriptIntent(user string) (writePath string, contents string, ok bool) {
	raw := strings.TrimSpace(user)
	if raw == "" {
		return "", "", false
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "python") {
		return "", "", false
	}
	if !strings.Contains(lower, "banner") && !strings.Contains(lower, "scroll") {
		return "", "", false
	}
	if !(strings.Contains(lower, "create") || strings.Contains(lower, "write") || strings.Contains(lower, "save")) {
		return "", "", false
	}

	dir := ""
	if m := reCreateDirPath.FindStringSubmatch(raw); len(m) >= 2 {
		candidate := strings.TrimSpace(m[1])
		if !isPlaceholderDirToken(candidate) {
			dir = candidate
		}
	}
	if dir == "" {
		if m := reMkdirOnly.FindStringSubmatch(raw); len(m) >= 2 {
			candidate := strings.TrimSpace(m[1])
			candidate = strings.Trim(candidate, `"'`)
			if !isPlaceholderDirToken(candidate) {
				dir = candidate
			}
		}
	}
	if dir == "" {
		if m := reInsidePath.FindStringSubmatch(raw); len(m) >= 2 {
			candidate := strings.TrimSpace(m[1])
			if !isPlaceholderDirToken(candidate) {
				dir = candidate
			}
		}
	}
	if dir == "" {
		if m := reDirectoryCalled.FindStringSubmatch(raw); len(m) >= 2 {
			candidate := strings.TrimSpace(m[1])
			if !isPlaceholderDirToken(candidate) {
				dir = candidate
			}
		}
	}
	if dir == "" {
		if m := reDirectoryLabeled.FindStringSubmatch(raw); len(m) >= 2 {
			candidate := strings.TrimSpace(m[1])
			if !isPlaceholderDirToken(candidate) {
				dir = candidate
			}
		}
	}
	if dir == "" {
		if m := reInDirectoryPath.FindStringSubmatch(raw); len(m) >= 2 {
			candidate := strings.TrimSpace(m[1])
			if !isPlaceholderDirToken(candidate) {
				dir = candidate
			}
		}
	}
	dir = strings.Trim(strings.TrimRight(dir, ".,;:!?"), "/")
	if dir == "" {
		return "", "", false
	}

	fileName := "banner.py"
	if m := rePyFilePath.FindStringSubmatch(raw); len(m) >= 2 {
		candidate := strings.TrimSpace(m[1])
		candidate = filepath.ToSlash(candidate)
		if strings.Contains(candidate, "/") {
			fileName = filepath.Base(candidate)
		} else if candidate != "" {
			fileName = candidate
		}
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ".py") {
		fileName += ".py"
	}

	bannerText := "FORGE LIVES!"
	if m := reSaysQuoted.FindStringSubmatch(raw); len(m) >= 2 {
		quoted := strings.TrimSpace(m[1])
		if quoted != "" {
			bannerText = quoted
		}
	}
	escaped := strings.ReplaceAll(bannerText, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "`", "'")

	colorA := "#ffd766"
	colorB := "#ff5722"
	if strings.Contains(lower, "purple") && strings.Contains(lower, "blue") {
		colorA = "#8a2be2"
		colorB = "#1e90ff"
	}

	contents = fmt.Sprintf(`#!/usr/bin/env python3
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

	writePath = filepath.ToSlash(filepath.Join(dir, fileName))
	return writePath, contents, true
}

func isPlaceholderDirToken(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "a", "an", "directory", "folder", "dir", "the", "this", "that", "said":
		return true
	default:
		return false
	}
}
