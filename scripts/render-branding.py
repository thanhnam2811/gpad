#!/usr/bin/env python3
"""Regenerate committed raster assets from SVG with librsvg (Ubuntu python3-gi/cairo)."""
from pathlib import Path
import shutil
import gi
import cairo

gi.require_version("Rsvg", "2.0")
from gi.repository import Rsvg  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


def render(source, destination, width, height):
    handle = Rsvg.Handle.new_from_file(str(source))
    surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, width, height)
    context = cairo.Context(surface)
    viewport = Rsvg.Rectangle()
    viewport.x, viewport.y, viewport.width, viewport.height = 0, 0, width, height
    handle.render_document(context, viewport)
    surface.write_to_png(str(destination))


def main():
    icons = ROOT / "assets/icons"
    icons.mkdir(exist_ok=True)
    source = ROOT / "assets/icon.svg"
    for size in (16, 24, 32, 48, 64, 128, 256, 512):
        render(source, icons / f"{size}.png", size, size)
    shutil.copyfile(source, ROOT / "internal/ui/icon.svg")
    shutil.copyfile(icons / "128.png", ROOT / "internal/ui/icon.png")
    shutil.copyfile(icons / "512.png", ROOT / "assets/icon.png")
    render(ROOT / "assets/branding/social-preview.svg",
           ROOT / "assets/branding/social-preview.png", 1280, 640)
    # Review sheet uses actual pixel sizes on both backgrounds; it is not a UI screenshot.
    surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, 1000, 480)
    ctx = cairo.Context(surface)
    for row, (background, foreground) in enumerate((("#f4f6f8", "#192633"), ("#111820", "#f3f6f8"))):
        def color(value):
            ctx.set_source_rgb(*(int(value[i:i+2], 16) / 255 for i in (1, 3, 5)))
        color(background)
        ctx.rectangle(0, row*240, 1000, 240)
        ctx.fill()
        for column, size in enumerate((16, 24, 32, 48, 64, 128)):
            x, y = 30 + column*148, row*240 + 48
            ctx.set_source_surface(cairo.ImageSurface.create_from_png(str(icons / f"{size}.png")), x, y)
            ctx.paint()
            color(foreground)
            ctx.set_font_size(13)
            ctx.move_to(x, row*240 + 210)
            ctx.show_text(f"{size} px")
    surface.write_to_png(str(ROOT / "assets/branding/icon-preview.png"))
    print("Rendered eight icon sizes, embedded icon copies, review sheet and social preview.")


if __name__ == "__main__":
    main()
