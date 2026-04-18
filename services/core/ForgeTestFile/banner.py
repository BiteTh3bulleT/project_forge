#!/usr/bin/env python3
"""
FORGE scrolling banner.
Attempts "Vegas Lights" first and falls back to system fonts if unavailable.
"""

import tkinter as tk
from tkinter import font as tkfont

TEXT = "FORGE LIVES!"
PRIMARY_FONT = "Vegas Lights"
FALLBACK_FONTS = ("Vegas Lights", "Arial Black", "Helvetica", "TkDefaultFont")
FONT_SIZE = 48
SPEED_PX = 3
FRAME_MS = 20

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
    fill="#ffd766",
    font=(font_family, FONT_SIZE, "bold"),
    anchor="w",
)

shadow_id = canvas.create_text(
    0,
    0,
    text=TEXT,
    fill="#ff5722",
    font=(font_family, FONT_SIZE, "bold"),
    anchor="w",
)


def layout():
    w = canvas.winfo_width()
    h = canvas.winfo_height()
    y = h // 2
    canvas.coords(shadow_id, w + 4, y + 3)
    canvas.coords(text_id, w, y)


def tick():
    canvas.move(text_id, -SPEED_PX, 0)
    canvas.move(shadow_id, -SPEED_PX, 0)
    bbox = canvas.bbox(text_id)
    if bbox and bbox[2] < 0:
        w = canvas.winfo_width()
        y = canvas.winfo_height() // 2
        canvas.coords(text_id, w, y)
        canvas.coords(shadow_id, w + 4, y + 3)
    root.after(FRAME_MS, tick)


root.bind("<Configure>", lambda _evt: layout())
root.after(100, layout)
root.after(FRAME_MS, tick)
root.mainloop()
