#!/usr/bin/env python3
"""Negative artifact checks using only synthetic archives and temporary paths."""
import hashlib
import io
import pathlib
import os
import shutil
import subprocess
import tarfile
import tempfile
import unittest

VERIFY = pathlib.Path(__file__).resolve().with_name("verify-release.sh")


class ArchiveGuards(unittest.TestCase):
    def run_archive(self, member, corrupt=False):
        with tempfile.TemporaryDirectory(prefix="gpad-archive-guard-") as directory:
            archive = pathlib.Path(directory) / "gpad-0.1.0-dev-linux-amd64.tar.gz"
            with tarfile.open(archive, "w:gz") as writer:
                entry = tarfile.TarInfo(member)
                if member.endswith("link"):
                    entry.type = tarfile.SYMTYPE
                    entry.linkname = "/etc/passwd"
                    writer.addfile(entry)
                else:
                    content = b"synthetic content"
                    entry.size = len(content)
                    writer.addfile(entry, io.BytesIO(content))
            digest = hashlib.sha256(archive.read_bytes()).hexdigest()
            archive.with_name(archive.name[:-7] + ".sha256").write_text(digest + "  " + archive.name + "\n")
            if corrupt:
                with archive.open("ab") as writer:
                    writer.write(b"damage")
            result = subprocess.run(["bash", str(VERIFY), str(archive)], capture_output=True, text=True, timeout=15)
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
            return result.stdout + result.stderr

    def test_corrupt_checksum(self):
        self.assertIn("FAILED", self.run_archive("root/synthetic", corrupt=True))

    def test_parent_escape(self):
        self.assertIn("Unsafe archive member", self.run_archive("../escape"))

    def test_absolute_path(self):
        self.assertIn("Unsafe archive member", self.run_archive("/escape"))

    def test_symlink(self):
        self.assertIn("Unsafe archive member", self.run_archive("root/link"))


class InstallerGuards(unittest.TestCase):
    def test_directory_target_is_rejected(self):
        with tempfile.TemporaryDirectory(prefix="gpad-install-guard-") as directory:
            root = pathlib.Path(directory)
            source = root / "source release"
            (source / "bin").mkdir(parents=True)
            (source / "share").mkdir()
            shutil.copyfile("/bin/true", source / "bin/gpad")
            target = root / "dest/usr/bin/gpad"
            target.mkdir(parents=True)
            result = subprocess.run(["bash", str(VERIFY.with_name("install.sh")), str(source)],
                                    env={**os.environ, "PREFIX": "/usr", "DESTDIR": str(root / "dest")},
                                    capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(list(target.iterdir()), [])
            self.assertEqual(list(target.parent.glob(".gpad-install.*")), [])

    def test_running_binary_can_be_replaced(self):
        with tempfile.TemporaryDirectory(prefix="gpad-install-running-") as directory:
            root = pathlib.Path(directory)
            source = root / "source"
            (source / "bin").mkdir(parents=True)
            (source / "share").mkdir()
            shutil.copyfile("/bin/true", source / "bin/gpad")
            target = root / "dest/usr/bin/gpad"
            target.parent.mkdir(parents=True)
            shutil.copy2("/bin/sleep", target)
            running = subprocess.Popen([str(target), "15"])
            try:
                subprocess.run(["bash", str(VERIFY.with_name("install.sh")), str(source)],
                               env={**os.environ, "PREFIX": "/usr", "DESTDIR": str(root / "dest")},
                               capture_output=True, text=True, check=True)
                self.assertIsNone(running.poll())
                self.assertEqual(target.read_bytes(), pathlib.Path("/bin/true").read_bytes())
                self.assertEqual(list(target.parent.glob(".gpad-install.*")), [])
            finally:
                running.terminate()
                running.wait(timeout=5)


if __name__ == "__main__":
    unittest.main()
