#!/usr/bin/env python3
"""Measure Gopad++ against the other text editors installed on this machine.

    scripts/bench-editors.py [file]        # default: scratch/test_500m.log

Every editor is launched twice, in its own process group, on a headless X
display (Xvfb, software GL) so nothing pops up on your desktop, and sampled
from /proc:

  idle    started with no file; the session ends once the resident set has
          stopped growing. Reports "start" (last growth of the resident set
          after launch, which is the editor finishing its startup) and the
          idle resident set.
  load    started with the large file. The read phase is the rchar growth
          phase: "ingest" is the wall time from rchar passing 4MB to its last
          growth on the file, i.e. how long the editor spent reading it.
          Gopad++ never reads its file - it maps it - so its ingest is the
          traced open()->mmap() syscall pair instead. The resident set is
          sampled for 6s after the read phase and reported at its peak.

Read the numbers with the machine and file that produced them. Each session is
killed (SIGTERM, then SIGKILL) and must leave no process behind: a GApplication
like Mousepad that is left running would swallow the next launch through D-Bus
and the numbers would be of the stub. Close your own editors first; the script
skips one that is already running.
"""
import argparse
import json
import os
import signal
import subprocess
import sys
import tempfile
import time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SAMPLE = 0.1          # seconds between /proc samples
RCHAR_START = 4 << 20  # bytes: the file read has started once reads pass 4MB
READ_IDLE = 1.5       # seconds without growth that end the read phase
SETTLE = 6.0          # seconds sampled after the read phase, for the peak RSS


def editors():
    return [
        ("Gopad++", [os.path.join(ROOT, "bin", "gpad")]),
        ("Mousepad", ["mousepad"]),
        ("Gedit", ["gedit"]),
        ("GNOME Text Editor", ["gnome-text-editor"]),
        ("VS Code", ["code"]),
    ]


def read_comm(pid):
    try:
        with open(f"/proc/{pid}/comm") as f:
            return f.read().strip()
    except OSError:
        return ""


def group_pids(pgid):
    pids = []
    for entry in os.listdir("/proc"):
        if not entry.isdigit():
            continue
        try:
            with open(f"/proc/{entry}/stat") as f:
                stat = f.read()
        except OSError:
            continue
        # pid (comm) state ppid pgrp session ... ; comm may hold spaces/parens.
        fields = stat[stat.rfind(")") + 2:].split()
        if len(fields) > 2 and fields[2] == str(pgid) and fields[0] != "Z":
            pids.append(int(entry))
    return pids


def group_rss_kb(pgid):
    total = 0
    for pid in group_pids(pgid):
        try:
            with open(f"/proc/{pid}/status") as f:
                for line in f:
                    if line.startswith("VmRSS:"):
                        total += int(line.split()[1])
                        break
        except OSError:
            continue
    return total


def group_rchar(pgid):
    total = 0
    for pid in group_pids(pgid):
        try:
            with open(f"/proc/{pid}/io") as f:
                for line in f:
                    if line.startswith("rchar:"):
                        total += int(line.split()[1])
                        break
        except OSError:
            continue
    return total


def env_for(argv, tag="", display=None):
    env = dict(os.environ)
    if display:
        env["DISPLAY"] = display
    if os.path.basename(argv[0]) == "gpad":
        # Never touch the user's real session; the single-instance socket
        # lives in XDG_RUNTIME_DIR, and a fresh pair per run rules out
        # forwarding this launch into a leftover instance.
        base = tempfile.mkdtemp(prefix=f"gpad-bench-{tag}-")
        env["HOME"] = os.path.join(base, "home")
        env["XDG_RUNTIME_DIR"] = os.path.join(base, "run")
        os.makedirs(env["HOME"], exist_ok=True)
        os.makedirs(env["XDG_RUNTIME_DIR"], mode=0o700, exist_ok=True)
    return env


class Session:
    """One measured editor launch."""

    def __init__(self, argv, env, args, name):
        self.name = name
        self.start = time.monotonic()
        self.proc = subprocess.Popen(argv + args, env=env, start_new_session=True,
                                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.pgid = self.proc.pid
        self.samples = []  # (t, rss_kb, rchar)

    def sample(self):
        rss, rchar = group_rss_kb(self.pgid), group_rchar(self.pgid)
        now = time.monotonic()
        self.samples.append((now, rss, rchar))
        return now, rss, rchar

    def rss_kb(self):
        return max((s[1] for s in self.samples), default=0)

    def idle(self, settle_s=2.0, max_s=30.0):
        """Sample until the resident set has held steady; the editor is up."""
        peak, last_growth = 0, self.start
        while True:
            now, rss, _ = self.sample()
            if rss > peak:
                peak, last_growth = rss, now
            if now - last_growth >= settle_s or now - self.start >= max_s:
                break
            time.sleep(SAMPLE)
        self.start_s = last_growth - self.start
        return self

    def load(self, max_s=90.0):
        """Sample the read phase of the file, then SETTLE seconds of it."""
        first_read = last_read = None
        finished, prev = False, 0
        while True:
            now, _, rchar = self.sample()
            if rchar > RCHAR_START:
                first_read = first_read or now
                if rchar > prev:
                    last_read = now
            prev = max(prev, rchar)
            if first_read and last_read and now - last_read >= READ_IDLE:
                finished = True
                break
            if now - self.start >= max_s:
                break
            time.sleep(SAMPLE)
        self.ingest_s = (last_read - first_read) if first_read and last_read else None
        self.unfinished = first_read is not None and not finished
        until = time.monotonic() + SETTLE
        while time.monotonic() < until:
            self.sample()
            time.sleep(SAMPLE)
        return self

    def kill(self):
        """SIGTERM then SIGKILL; the group must be empty, or the next launch
        of a single-instance editor would land in the survivor."""
        for sig in (signal.SIGTERM, signal.SIGKILL):
            try:
                os.killpg(self.pgid, sig)
            except ProcessLookupError:
                pass
            deadline = time.time() + 5
            while time.time() < deadline and group_pids(self.pgid):
                time.sleep(0.1)
            if not group_pids(self.pgid):
                break
        else:
            print(f"# WARNING: {self.name} left processes behind", file=sys.stderr)
        try:
            self.proc.wait(timeout=5)  # reap it, or it stays a zombie
        except subprocess.TimeoutExpired:
            pass


def mmap_ingest(binary, path, display):
    """The traced open()->mmap() pair: the ingest time of a mapping editor."""
    trace = tempfile.mktemp(prefix="gpad-bench-", suffix=".trace")
    size = os.path.getsize(path)
    proc = subprocess.Popen(["strace", "-f", "-ttt", "-o", trace, "-e", "trace=openat,mmap,read",
                             binary, path], env=env_for([binary], "mmap", display),
                            start_new_session=True,
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(4)
    try:
        os.killpg(proc.pid, signal.SIGTERM)
        time.sleep(0.5)
        os.killpg(proc.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass
    proc.wait()
    opened = mapped = None
    for line in open(trace, errors="replace"):
        parts = line.split(None, 2)
        if len(parts) < 3:
            continue
        try:
            ts = float(parts[1])
        except ValueError:
            continue
        if opened is None and path in line and "openat(" in line:
            opened = ts
        elif opened and "mmap(" in line and f", {size}," in line and "MAP_SHARED" in line:
            mapped = ts
            break
    return (mapped - opened) if opened and mapped else None


def running(binary):
    """Whether an editor of this name is already up, whose single-instance
    launch would swallow ours and leave us measuring the stub."""
    comm = os.path.basename(binary)[:15]  # /proc/<pid>/comm is 15 bytes
    return any(read_comm(pid) == comm for pid in os.listdir("/proc") if pid.isdigit())


def headless_display(number=99):
    """Start Xvfb, so the editors open off-screen; returns its DISPLAY."""
    if os.path.exists(f"/tmp/.X11-unix/X{number}"):
        number += 50
    proc = subprocess.Popen(["Xvfb", f":{number}", "-screen", "0", "1920x1080x24", "-nolisten", "tcp"],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                            start_new_session=True)
    display = f":{number}"
    for _ in range(50):
        if subprocess.run(["xdpyinfo", "-display", display],
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
            print(f"# headless display {display} (Xvfb pid {proc.pid})", file=sys.stderr)
            return display, proc
        time.sleep(0.2)
    raise SystemExit("Xvfb did not come up; pass --display :0 to use the desktop")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("file", nargs="?", default=os.path.join(ROOT, "scratch", "test_500m.log"))
    ap.add_argument("--only", help="run one editor by name")
    ap.add_argument("--display", help="X display to run on (default: a headless Xvfb)")
    ap.add_argument("--max-load", type=float, default=240.0,
                    help="seconds a load session may spend reading the file")
    args = ap.parse_args()

    xvfb = None
    if args.display:
        display = args.display
    else:
        display, xvfb = headless_display()
    path = os.path.abspath(args.file)
    size_mb = os.path.getsize(path) / (1 << 20)
    print(f"# {os.path.basename(path)} ({size_mb:.0f} MB) on {os.uname().nodename}, {display}\n")
    print("| Editor | start | idle RSS | ingest | RSS loaded |")
    print("|---|---:|---:|---:|---:|")
    rows = []
    for name, argv in editors():
        if args.only and args.only.lower() not in name.lower():
            continue
        if not os.path.exists(argv[0]) and "/" in argv[0]:
            print(f"| {name} | build bin/gpad first | | | |")
            continue
        # gpad is measured with a private HOME and runtime dir (env_for), so a
        # running gpad cannot swallow this launch; every other editor here
        # answers to a global name on the session bus instead.
        if os.path.basename(argv[0]) != "gpad" and running(argv[0]):
            print(f"| {name} | already running, skipped | | | |")
            continue
        print(f"# measuring {name} ({' '.join(argv)})", file=sys.stderr)
        idle = Session(argv, env_for(argv, "idle", display), [], name).idle()
        idle.kill()
        load = Session(argv, env_for(argv, "load", display), [path], name).load(args.max_load)
        load.kill()
        if idle.rss_kb() == 0 or load.rss_kb() == 0:
            print(f"| {name} | did not start | | | |")
            continue
        ingest = (mmap_ingest(argv[0], path, display) if os.path.basename(argv[0]) == "gpad"
                  else load.ingest_s)
        cell = ingest_cell(ingest, load)
        rows.append({
            "editor": name,
            "start_s": round(idle.start_s, 2),
            "idle_rss_mb": round(idle.rss_kb() / 1024, 1),
            "ingest_s": round(ingest, 6) if ingest else None,
            "ingest_unfinished": bool(getattr(load, "unfinished", False)),
            "load_rss_mb": round(load.rss_kb() / 1024, 1),
        })
        print(f"| {name} | {rows[-1]['start_s']} s | {rows[-1]['idle_rss_mb']} MB | {cell} | "
              f"{rows[-1]['load_rss_mb']} MB |")
    if xvfb:
        xvfb.terminate()
    out = "/tmp/opencode/bench/results.json"
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "w") as f:
        json.dump({"file": path, "size_mb": size_mb, "rows": rows}, f, indent=2)
    print(f"\n# raw: {out}\n")
    print(json.dumps(rows, indent=2))


def ingest_cell(ingest, load):
    if not ingest:
        return "no bytes read (maps)"
    if ingest < 1:
        return f"{ingest * 1000:.2f} ms"
    if getattr(load, "unfinished", False):
        return f">{ingest:.0f} s (unfinished)"
    return f"{ingest:.2f} s"


if __name__ == "__main__":
    main()
