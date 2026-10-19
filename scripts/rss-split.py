#!/usr/bin/env python3
"""Idle and loaded RSS of one editor, split into private vs file-backed.

`bench-editors.py` reports VmRSS, which lumps together an editor's own heap and
the shared library code it maps (GTK, the GL stack). That hides the number that
matters when the machine is under pressure: RssAnon, the memory this process
alone owns. This reads /proc/<pid>/status instead, under the same headless
conditions (Xvfb, one process group per editor, isolated HOME and runtime dir
for Gopad++, so the user's session is never touched).

    python3 scripts/rss-split.py [file ...]
"""
import os
import signal
import subprocess
import sys
import tempfile
import time

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
EDITORS = [("Gopad++", [os.path.join(REPO, "bin", "gpad")]), ("Mousepad", ["mousepad"])]
DEFAULT_FILE = os.path.join(REPO, "scratch", "test_500m.log")


def xvfb():
    """Start a private headless display, so no window opens on the desktop."""
    for n in range(98, 199):
        if os.path.exists(f"/tmp/.X11-unix/X{n}"):
            continue
        p = subprocess.Popen(["Xvfb", f":{n}", "-screen", "0", "1920x1080x24"],
                             stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                             start_new_session=True)
        for _ in range(100):
            if os.path.exists(f"/tmp/.X11-unix/X{n}"):
                return p, f":{n}"
            time.sleep(0.1)
        p.kill()
    raise SystemExit("no free X display")


def rss(pid):
    out = {}
    try:
        with open(f"/proc/{pid}/status") as f:
            for line in f:
                if line.startswith(("VmRSS", "RssAnon", "RssFile", "RssShmem")):
                    k, v = line.split(":")
                    out[k] = int(v.split()[0]) / 1024  # MB
    except FileNotFoundError:
        pass
    return out


def env_for(argv, display):
    env = dict(os.environ)
    env["DISPLAY"] = display
    if os.path.basename(argv[0]) == "gpad":
        # A private HOME and runtime dir: the single-instance socket lives in
        # XDG_RUNTIME_DIR, so this can never reach the user's running editor.
        base = tempfile.mkdtemp(prefix="rss-split-")
        env["HOME"] = os.path.join(base, "home")
        env["XDG_RUNTIME_DIR"] = os.path.join(base, "run")
        os.makedirs(env["HOME"], exist_ok=True)
        os.makedirs(env["XDG_RUNTIME_DIR"], mode=0o700, exist_ok=True)
    return env


def measure(argv, display, load):
    """Return the settled RSS sample with and without the file open."""
    p = subprocess.Popen(argv, env=env_for(argv, display), start_new_session=True,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        last, settled = 0.0, 0
        while settled < 8:  # wait for the editor to stop growing
            time.sleep(0.5)
            cur = rss(p.pid).get("VmRSS", 0.0)
            settled = settled + 1 if abs(cur - last) < 0.5 else 0
            last = cur
        idle = rss(p.pid)
        if load:
            time.sleep(6)  # the same grace the installer of the bench gives
            for _ in range(40):
                time.sleep(0.5)
                cur = rss(p.pid).get("VmRSS", 0.0)
                if abs(cur - last) < 0.5:
                    break
                last = cur
        return idle, rss(p.pid)
    finally:
        for sig in (signal.SIGTERM, signal.SIGKILL):
            try:
                os.killpg(p.pid, sig)
                p.wait(timeout=10)
                break
            except Exception:
                pass


def main():
    files = sys.argv[1:] or [DEFAULT_FILE]
    if not os.path.exists(files[0]):
        raise SystemExit(f"missing file: {files[0]}")
    xv, display = xvfb()
    try:
        for label, argv in EDITORS:
            for f in [None] + files:
                idle, loaded = measure(argv + ([f] if f else []), display, f is not None)
                got = loaded if f else idle
                print(f"{label:<10} {'idle' if not f else os.path.basename(f):<12} "
                      f"VmRSS {got.get('VmRSS', 0):7.1f} MB  "
                      f"anon {got.get('RssAnon', 0):7.1f}  file {got.get('RssFile', 0):7.1f}  "
                      f"shmem {got.get('RssShmem', 0):5.1f}")
                sys.stdout.flush()
    finally:
        xv.terminate()


if __name__ == "__main__":
    main()
