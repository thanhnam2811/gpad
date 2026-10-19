#!/usr/bin/env python3
"""Does opening a file bring the running gpad window forward on Wayland?

A file manager or the shell starts gpad with a file and gives it an activation
token for the launch it asked for; the running instance has to pass that token
on with the path, and present its window with it. Wayland has no "raise
yourself" call: the compositor focuses a window only when it presents with the
token of that launch.

This starts a server instance on the live session and a GTK4 launcher in front
of it (what Nautilus is), which hands a second, forwarding instance the token
exactly as a launcher does, then watches its own window lose the focus.

Three things are reported:

  sent       the token reached the server (the point of the fix)
  activation the server's Wayland connection asked the compositor to activate
             its window with it (what the compositor needs)
  focus      the launcher lost the focus to it, as the user would see

The first two are gpad's doing and must always hold. The third is the
compositor's, reported as information: it accepts an activation token whose
issuer still holds the seat's last grab, and a script's launcher window has
taken its focus from no click of the user, so it can refuse. Run this from the
window whose focus you want to lose (a terminal, a file manager) for a faithful
answer, and check the exit status: it is non-zero when gpad failed to ask.

    scratch/focus-check.py            # with the token
    scratch/focus-check.py --no-token # without it: nothing to activate with
"""
import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GPAD = os.path.join(ROOT, "bin", "gpad")
FILE = os.path.join(ROOT, "cmd", "gpad", "main.go")

LAUNCHER = r"""
import os, subprocess, sys
import gi
gi.require_version("Gtk", "4.0")
gi.require_version("Gio", "2.0")
from gi.repository import Gtk, Gio, GLib

gpad, target = sys.argv[1], sys.argv[2]

def on_activate(app):
    win = Gtk.ApplicationWindow(application=app, title="gpad focus check")
    win.set_default_size(240, 90)
    win.present()

    def launch():
        env = dict(os.environ)
        token = ""
        if not env.pop("NO_TOKEN", None):
            info = Gio.AppInfo.create_from_commandline(gpad, "gpad",
                                                       Gio.AppInfoCreateFlags.NONE)
            ctx = win.get_display().get_app_launch_context()
            token = ctx.get_startup_notify_id(info, []) or ""
            # What a launcher hands over: the shell sets XDG_ACTIVATION_TOKEN,
            # GLib applications set DESKTOP_STARTUP_ID.
            env["XDG_ACTIVATION_TOKEN"] = token
            env["DESKTOP_STARTUP_ID"] = token
        print("token=%s" % token, flush=True)
        subprocess.Popen([gpad, target], env=env)
        left = []
        def watch(*_):
            # The launcher window losing the focus is the user-visible result.
            if not win.is_active():
                print("focus-leaving=yes", flush=True)
                app.quit()
                return False
            left.append(1)
            if len(left) > 40:
                print("focus-leaving=no", flush=True)
                app.quit()
                return False
            return True
        GLib.timeout_add(100, watch)
        return False

    GLib.timeout_add(1500, launch)

app = Gtk.Application(application_id="dev.gpad.focuscheck")
app.connect("activate", on_activate)
app.run([])
"""


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--no-token", action="store_true", help="forward no activation token")
    args = ap.parse_args()

    work = tempfile.mkdtemp(prefix="gpad-focus-")
    runtime, home = os.path.join(work, "rt"), os.path.join(work, "home")
    # 0700: libwayland refuses a runtime dir anyone else can look into, and the
    # instance would fall back to X11, where this all means something else.
    os.makedirs(runtime, mode=0o700)
    os.makedirs(home, mode=0o700)
    # A real Wayland client of the session that owns its runtime dir: sharing
    # it would forward into the editor the user has open.
    os.symlink(os.path.join(os.environ["XDG_RUNTIME_DIR"], "wayland-0"),
               os.path.join(runtime, "wayland-0"))

    env = dict(os.environ)
    env["HOME"] = home
    env["XDG_RUNTIME_DIR"] = runtime
    env["WAYLAND_DISPLAY"] = "wayland-0"
    env["GDK_BACKEND"] = "wayland"  # X11 is not the thing being checked
    env["WAYLAND_DEBUG"] = "1"      # the wire is the evidence below
    if args.no_token:
        env["NO_TOKEN"] = "1"

    log_path = os.path.join(work, "server.log")
    with open(log_path, "w") as log:
        server = subprocess.Popen([GPAD], env=env, stdout=log, stderr=subprocess.STDOUT)
        try:
            for _ in range(80):
                if os.path.exists(os.path.join(runtime, "gpad.sock")):
                    break
                time.sleep(0.25)
            if server.poll() is not None:
                print("the server instance did not start:", open(log_path).read())
                return 1
            time.sleep(3)  # let its window have the focus, to lose it
            launcher = os.path.join(work, "launcher.py")
            open(launcher, "w").write(LAUNCHER)
            run = subprocess.run(["python3", launcher, GPAD, FILE], env=env,
                                 text=True, capture_output=True, timeout=90)
            time.sleep(1)
        finally:
            server.terminate()
            try:
                server.wait(timeout=10)
            except subprocess.TimeoutExpired:
                server.kill()

    trace = open(log_path).read()
    token = ""
    for line in run.stdout.splitlines():
        if line.startswith("token="):
            token = line[len("token="):]
    if run.returncode != 0 or not token and not args.no_token:
        print("the launcher failed:", run.stdout, run.stderr)
        return 1

    sent = bool(token) and token in trace
    activates = re.findall(r'xdg_activation_v1@\d+\.activate\("([^"]+)"', trace)
    activation = token in activates
    focus = "focus-leaving=yes" in run.stdout

    print(f"token out of the launcher: {token or '(none)'}")
    print(f"sent       : {sent or args.no_token}  (the server was given the token)")
    print(f"activation : {activation or args.no_token}  "
          f"(asked to activate with it: {sorted(set(activates))})")
    print(f"focus      : {focus}  (informational: what the compositor decided)")
    ok = sent == (not args.no_token) and activation == (not args.no_token)
    if not ok:
        print("\nthe server never activated with the token it was given:")
        print(*[l for l in trace.splitlines() if "xdg_activation" in l], sep="\n")
    if os.environ.get("KEEP"):
        print("work dir:", work)
    else:
        shutil.rmtree(work, ignore_errors=True)
    return 0 if ok else 1


sys.exit(main())
