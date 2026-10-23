#!/usr/bin/env python3
"""Copy module license notices with stable module-relative names and manifests."""
import json
import os
import pathlib
import shutil
import subprocess
import sys

output = pathlib.Path(sys.argv[1])
raw = subprocess.check_output(["go", "list", "-tags", "mygo_noinspector", "-deps", "-json", "./cmd/gpad"], text=True, env={**os.environ, "CGO_ENABLED": "0", "GOOS": "linux"})
decoder = json.JSONDecoder()
modules = []
seen = set()
while raw.strip():
    package, end = decoder.raw_decode(raw.lstrip())
    raw = raw.lstrip()[end:]
    module = package.get("Module")
    if not module or module.get("Main") or module["Path"] in seen:
        continue
    seen.add(module["Path"])
    resolved = module.get("Replace", module)
    source = pathlib.Path(resolved["Dir"])
    files = sorted(p for p in source.iterdir() if p.is_file() and p.name.upper().startswith(("LICENSE", "COPYING", "NOTICE", "AUTHORS", "PATENTS")))
    if not any(p.name.upper().startswith(("LICENSE", "COPYING")) for p in files):
        raise SystemExit(f"Missing license: {module['Path']}")
    destination = output / module["Path"].replace("/", "_")
    destination.mkdir()
    for file in files:
        shutil.copyfile(file, destination / file.name)
    modules.append({"module": module["Path"], "version": module.get("Version", ""), "replacement": resolved.get("Path", ""), "notices": [p.name for p in files]})
(output / "dependencies.json").write_text(json.dumps(modules, indent=2) + "\n")

# Preserve nested notices in the vendored framework (including shader/runtime code).
vendored = pathlib.Path("third_party/mygo")
for source in sorted(vendored.rglob("LICENSE*")):
    if source.is_file():
        destination = output / "vendored-mygo" / source.relative_to(vendored)
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, destination)
