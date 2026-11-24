"""Send native keys to the currently focused window on a task-owned X display."""
import ctypes as C
import sys
import time

x = C.CDLL('libX11.so.6')
t = C.CDLL('libXtst.so.6')
x.XOpenDisplay.argtypes = [C.c_char_p]
x.XOpenDisplay.restype = C.c_void_p
x.XStringToKeysym.argtypes = [C.c_char_p]
x.XStringToKeysym.restype = C.c_ulong
x.XKeysymToKeycode.argtypes = [C.c_void_p, C.c_ulong]
x.XKeysymToKeycode.restype = C.c_uint
x.XFlush.argtypes = [C.c_void_p]
x.XCloseDisplay.argtypes = [C.c_void_p]
t.XTestFakeKeyEvent.argtypes = [C.c_void_p, C.c_uint, C.c_int, C.c_ulong]
display = x.XOpenDisplay(None)
if not display:
    raise SystemExit('Task display unavailable')
try:
    for name in sys.argv[1:]:
        code = x.XKeysymToKeycode(display, x.XStringToKeysym(name.encode()))
        t.XTestFakeKeyEvent(display, code, 1, 0)
        t.XTestFakeKeyEvent(display, code, 0, 0)
        x.XFlush(display)
        time.sleep(.1)
finally:
    x.XCloseDisplay(display)
