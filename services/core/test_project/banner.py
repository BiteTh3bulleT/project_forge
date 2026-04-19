#!/usr/bin/env python3
"""
FORGE scrolling banner.
Attempts "Vegas Lights" first and falls back to system fonts if unavailable.
"""

import math
import tkinter as tk
from tkinter import font as tkfont

TEXT = "Te Queiro Mucho Mi Riena"
PRIMARY_FONT = "Vegas Lights"
FALLBACK_FONTS = ("Vegas Lights", "Arial Black", "Helvetica", "TkDefaultFont")
FONT_SIZE = 48
SPEED_PX = 3
FRAME_MS = 20
PULSE_MS = 25
PULSE_STEP = 0.09
COLOR_A = "#8a2be2"
COLOR_B = "#1e90ff"

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
