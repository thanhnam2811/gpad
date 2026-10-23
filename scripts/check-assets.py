#!/usr/bin/env python3
"""Check committed branding/metadata sources without image-rendering dependencies."""
from pathlib import Path
import struct
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parents[1]


def require(condition, message):
    if not condition:
        raise SystemExit(message)


def png_size(path):
    data = path.read_bytes()
    require(data[:8] == b"\x89PNG\r\n\x1a\n" and data[12:16] == b"IHDR", f"invalid PNG: {path}")
    return struct.unpack(">II", data[16:24])


def main():
    for size in (16, 24, 32, 48, 64, 128, 256, 512):
        require(png_size(ROOT / f"assets/icons/{size}.png") == (size, size), f"wrong icon size: {size}")
    for generated, canonical in (("internal/ui/icon.svg", "assets/icon.svg"),
                                  ("internal/ui/icon.png", "assets/icons/128.png"),
                                  ("assets/icon.png", "assets/icons/512.png")):
        require((ROOT / generated).read_bytes() == (ROOT / canonical).read_bytes(),
                f"stale generated asset: {generated}; run render-branding.py")
    require(png_size(ROOT / "assets/branding/social-preview.png") == (1280, 640), "wrong social preview size")
    for path in (ROOT / "assets").rglob("*.svg"):
        tree = ET.parse(path)
        for element in tree.iter():
            require(element.tag.rsplit("}", 1)[-1] not in ("script", "foreignObject"), f"active SVG: {path}")
            for key, value in element.attrib.items():
                if key.rsplit("}", 1)[-1] == "href":
                    require(value.startswith("#"), f"external SVG resource: {path}")
    desktop = (ROOT / "packaging/gpad.desktop").read_text()
    require("Name=Gopad++\n" in desktop and "Icon=gpad\n" in desktop, "desktop naming mismatch")
    component = ET.parse(ROOT / "packaging/io.github.thanhnam2811.gpad.metainfo.xml").getroot()
    require(component.findtext("id") == "io.github.thanhnam2811.gpad", "AppStream ID mismatch")
    require(component.findtext("launchable") == "gpad.desktop", "AppStream launcher mismatch")
    print("Branding dimensions, embedded copies, SVG resources and desktop identity: PASS")


if __name__ == "__main__":
    main()
