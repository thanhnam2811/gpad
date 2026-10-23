#!/usr/bin/env python3
"""Real X11 GUI smoke/capture in private HOME, XDG, D-Bus and Xvfb resources.

Requires Xvfb, dbus-run-session, xwininfo, ImageMagick import and libXtst.
Never connect this helper to the user's display. Run: python3 scripts/gui-smoke.py BINARY OUTDIR
"""
import argparse
import hashlib
import ctypes as C
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import time


def wait_for(check, description, timeout=12):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        result = check()
        if result:
            return result
        time.sleep(0.1)
    raise RuntimeError(f"timed out: {description}")


def child(binary, output, theme):
    # The parent supplies a fresh display and a fresh session bus.
    if not os.environ.get("GPAD_PRIVATE_GUI"):
        raise RuntimeError("private GUI marker missing")
    subprocess.run(["gsettings", "set", "org.gnome.desktop.interface", "color-scheme", f"prefer-{theme}"], check=True)
    x = C.CDLL("libX11.so.6")
    xt = C.CDLL("libXtst.so.6")
    x.XOpenDisplay.restype = C.c_void_p
    x.XOpenDisplay.argtypes = [C.c_char_p]
    display = x.XOpenDisplay(None)
    if not display:
        raise RuntimeError("cannot open private X server")
    x.XStringToKeysym.argtypes = [C.c_char_p]
    x.XStringToKeysym.restype = C.c_ulong
    x.XKeysymToKeycode.argtypes = [C.c_void_p, C.c_ulong]
    x.XKeysymToKeycode.restype = C.c_uint
    x.XSetInputFocus.argtypes = [C.c_void_p, C.c_ulong, C.c_int, C.c_ulong]
    x.XSync.argtypes = [C.c_void_p, C.c_int]
    x.XCloseDisplay.argtypes = [C.c_void_p]
    xt.XTestFakeKeyEvent.argtypes = [C.c_void_p, C.c_uint, C.c_int, C.c_ulong]
    xt.XTestFakeMotionEvent.argtypes = [C.c_void_p, C.c_int, C.c_int, C.c_int, C.c_ulong]
    xt.XTestFakeButtonEvent.argtypes = [C.c_void_p, C.c_uint, C.c_int, C.c_ulong]

    def key(name, down):
        code = x.XKeysymToKeycode(display, x.XStringToKeysym(name.encode()))
        if not code:
            raise RuntimeError(f"unmapped key {name}")
        xt.XTestFakeKeyEvent(display, code, down, 0)

    def chord(*names):
        for name in names:
            key(name, 1)
        for name in reversed(names):
            key(name, 0)
        x.XSync(display, 0)
        time.sleep(0.2)

    def text(value):
        named = {" ": "space", "\n": "Return", "\t": "Tab"}
        shifted = '~!@#$%^&*()_+{}|:"<>?'
        for ch in value:
            shift = ch.isupper() or ch in shifted
            if shift:
                key("Shift_L", 1)
            key(named.get(ch, ch), 1)
            key(named.get(ch, ch), 0)
            if shift:
                key("Shift_L", 0)
            x.XSync(display, 0)
            time.sleep(0.025)
        time.sleep(0.3)

    def click(px, py):
        xt.XTestFakeMotionEvent(display, -1, px + origin[0], py + origin[1], 0)
        xt.XTestFakeButtonEvent(display, 1, 1, 0)
        xt.XTestFakeButtonEvent(display, 1, 0, 0)
        x.XSync(display, 0)
        time.sleep(0.3)

    def window():
        tree = subprocess.check_output(["xwininfo", "-root", "-tree"], text=True)
        found = re.search(r'(0x[0-9a-f]+) "Gopad\+\+"', tree)
        if not found:
            return None
        wid = int(found.group(1), 16)
        details = subprocess.check_output(["xwininfo", "-id", hex(wid)], text=True)
        return wid if "Map State: IsViewable" in details else None

    sample = Path(os.environ["HOME"]) / "workspace.go"
    sample.write_text('''// Gopad++ · isolated GUI smoke test
// Synthetic content. No personal files or sessions are used.
package main

import "fmt"

// A compact workspace for everyday text and code.
func main() {
    features := []string{
        "Tabs and keyboard shortcuts",
        "Find, replace, and go to line",
        "Private session recovery",
    }
    for _, feature := range features {
        fmt.Println(feature)
    }
}

// Save safely. Keep editing.
''')
    process = None
    log = (output / f"gui-{theme}.log").open("w")
    results = {"display": "private Xvfb / X11, software OpenGL", "theme": theme,
               "binary_sha256": hashlib.sha256(Path(binary).read_bytes()).hexdigest(),
               "version": subprocess.check_output([binary, "--version"], text=True).strip()}
    try:
        process = subprocess.Popen([binary, str(sample)], stdout=log, stderr=log)
        wid = wait_for(window, "GPAD window")
        geometry = subprocess.check_output(["xwininfo", "-id", hex(wid)], text=True)
        origin = tuple(int(re.search(r"Absolute upper-left " + axis + r":\s*(-?\d+)", geometry).group(1)) for axis in ("X", "Y"))
        x.XSetInputFocus(display, wid, 2, 0)
        x.XSync(display, 0)
        time.sleep(1)

        def capture(name):
            subprocess.run(["import", "-window", hex(wid), str(output / f"{name}-{theme}.png")], check=True)

        capture("editor")
        click(300, 226)
        results["startup"] = "PASS"
        results["window_class"] = subprocess.check_output(["xprop", "-id", hex(wid), "WM_CLASS"], text=True).strip()
        icon_property = subprocess.check_output(["xprop", "-id", hex(wid), "_NET_WM_ICON"], text=True)
        if "128 x 128" not in icon_property:
            raise RuntimeError("128 px native window icon is missing")
        results["window_icon"] = "PASS (128 x 128 _NET_WM_ICON)"
        chord("Control_L", "End")
        text("\nguismokesaveverified\n")
        chord("Control_L", "s")
        capture("save-check")
        wait_for(lambda: "guismokesaveverified" in sample.read_text(), "saved synthetic file")
        results["keyboard_edit_and_save"] = "PASS"
        chord("Control_L", "f")
        text("features")
        capture("find")
        chord("Escape")
        # Inspect the real File popover without relying on external clipboard tools.
        click(20, 11)
        capture("menu")
        chord("Escape")
        results["find_and_menu_capture"] = "PASS (visually inspect captures)"
        click(248, 11)
        click(290, 36)
        capture("about")
        chord("Escape")
        # A second process forwards into the existing instance and exits.
        forwarded = sample.with_name("forwarded.txt")
        forwarded.write_text("Single-instance forwarding on a private display.\n")
        subprocess.run([binary, str(forwarded)], check=True, timeout=8)
        time.sleep(0.5)
        results["single_instance_forwarding"] = "PASS (client returned; inspect session)"
        x.XSetInputFocus(display, wid, 2, 0)
        chord("Control_L", "n")
        click(275, 60)
        click(200, 100)
        text("recoveryprivatedraft")
        capture("draft")
        wal = Path(os.environ["HOME"]) / ".config/gpad/session/session.wal"
        try:
            wait_for(lambda: wal.exists() and b"recoveryprivatedraft" in wal.read_bytes(), "autosaved draft")
        except RuntimeError:
            if wal.exists():
                (output / "debug-private-session.wal").write_bytes(wal.read_bytes())
            raise
        process.kill()  # Crash only this private instance after autosave acknowledgement.
        process.wait(timeout=5)
        process = subprocess.Popen([binary], stdout=log, stderr=log)
        wid = wait_for(window, "recovery window")
        x.XSetInputFocus(display, wid, 2, 0)
        x.XSync(display, 0)
        time.sleep(1)
        capture("recovery")
        results["crash_recovery"] = "PASS (WAL draft + restored window; inspect capture)"
        chord("Control_L", "q")
        process.wait(timeout=8)
        if process.returncode:
            raise RuntimeError(f"GPAD quit status {process.returncode}")
        results["graceful_shutdown"] = "PASS"
        results["private_resources_only"] = True
        (output / f"gui-{theme}.json").write_text(json.dumps(results, indent=2) + "\n")
        print(json.dumps(results, indent=2))
    finally:
        if process and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        log.close()
        x.XCloseDisplay(display)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--theme", choices=("dark", "light"), default="dark")
    parser.add_argument("--private-child", action="store_true", help=argparse.SUPPRESS)
    args = parser.parse_args()
    binary, output = args.binary.resolve(), args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    if args.private_child:
        child(str(binary), output, args.theme)
        return
    for tool in ("Xvfb", "dbus-run-session", "xwininfo", "xprop", "import", "gsettings"):
        if not shutil.which(tool):
            parser.error(f"missing prerequisite: {tool}")
    with tempfile.TemporaryDirectory(prefix="gpad-private-gui-") as directory:
        root = Path(directory)
        for subdir in ("home", "runtime", "config", "cache", "data"):
            (root / subdir).mkdir(mode=0o700)
        # Whitelist environment; never inherit the active compositor/session bus.
        env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "LANG": "C.UTF-8",
               "HOME": str(root / "home"), "XDG_RUNTIME_DIR": str(root / "runtime"),
               "XDG_CONFIG_HOME": str(root / "config"), "XDG_CACHE_HOME": str(root / "cache"),
               "XDG_DATA_HOME": str(root / "data"), "GDK_BACKEND": "x11",
               "GSETTINGS_BACKEND": "keyfile", "LIBGL_ALWAYS_SOFTWARE": "1",
               "NO_AT_BRIDGE": "1", "GIO_USE_VFS": "local", "GTK_USE_PORTAL": "0",
               "GPAD_PRIVATE_GUI": "1"}
        bus_config = root / "bus.conf"
        bus_config.write_text('<busconfig><type>session</type><listen>unix:tmpdir=' + str(root / "runtime") + '</listen><auth>EXTERNAL</auth><policy context="default"><allow send_destination="*"/><allow receive_sender="*"/><allow own="*"/></policy></busconfig>')
        with (output / f"xvfb-{args.theme}.log").open("w") as log:
            xvfb = subprocess.Popen(["Xvfb", "-displayfd", "1", "-screen", "0", "1100x780x24",
                                     "-nolisten", "tcp", "-noreset"], stdout=subprocess.PIPE,
                                    stderr=log, text=True, env=env)
            try:
                number = xvfb.stdout.readline().strip()
                if not number.isdigit():
                    raise RuntimeError("Xvfb did not allocate a private display")
                env["DISPLAY"] = ":" + number
                subprocess.run(["dbus-run-session", "--config-file=" + str(bus_config), "--", sys.executable, str(Path(__file__).resolve()),
                                str(binary), str(output), "--theme", args.theme, "--private-child"],
                               env=env, check=True, timeout=90)
            finally:
                xvfb.terminate()
                xvfb.wait(timeout=5)


if __name__ == "__main__":
    main()
