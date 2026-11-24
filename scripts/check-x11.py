#!/usr/bin/env python3
"""Check the display tools used by native Chromium zoom tests, without a DB."""

import ctypes
import os
import select
import shutil
import subprocess


def main():
    for executable in ("Xvfb", "xdpyinfo", "xwininfo"):
        if shutil.which(executable) is None:
            raise SystemExit(f"Native zoom requires {executable}; install xvfb and x11-utils.")
    for library in ("libX11.so.6", "libXtst.so.6"):
        ctypes.CDLL(library)

    # Xvfb selects an unused display; only this process is reclaimed below.
    with subprocess.Popen(
        ["Xvfb", "-displayfd", "1", "-screen", "0", "1920x1200x24",
         "-nolisten", "tcp", "-nolisten", "unix"],
        stdout=subprocess.PIPE, text=True,
    ) as display:
        try:
            if not select.select([display.stdout], [], [], 5)[0]:
                raise SystemExit("Native zoom Xvfb display did not become ready within 5 seconds.")
            number = display.stdout.readline().strip()
            if not number.isdigit():
                raise SystemExit("Native zoom Xvfb exited without allocating a display.")
            env = dict(os.environ, DISPLAY=f":{number}")
            for command in (["xdpyinfo"], ["xwininfo", "-root", "-tree"]):
                subprocess.run(command, env=env, check=True, stdout=subprocess.DEVNULL, timeout=5)
            print("Native zoom display ready: Xvfb, xdpyinfo, xwininfo, libX11 and libXtst.")
        finally:
            display.terminate()
            try:
                display.wait(timeout=5)
            except subprocess.TimeoutExpired:
                display.kill()
                display.wait()


if __name__ == "__main__":
    main()
