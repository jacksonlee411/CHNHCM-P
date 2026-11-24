import ctypes as C,sys,time
x=C.CDLL('libX11.so.6');t=C.CDLL('libXtst.so.6'); P=C.c_void_p; U=C.c_ulong
x.XOpenDisplay.argtypes=[C.c_char_p];x.XOpenDisplay.restype=P
x.XDefaultRootWindow.argtypes=[P];x.XDefaultRootWindow.restype=U
x.XQueryTree.argtypes=[P,U,C.POINTER(U),C.POINTER(U),C.POINTER(C.POINTER(U)),C.POINTER(C.c_uint)]
x.XFetchName.argtypes=[P,U,C.POINTER(C.c_char_p)]
x.XSetInputFocus.argtypes=[P,U,C.c_int,U]
x.XStringToKeysym.argtypes=[C.c_char_p];x.XStringToKeysym.restype=U
x.XKeysymToKeycode.argtypes=[P,U];x.XKeysymToKeycode.restype=C.c_uint
x.XFlush.argtypes=[P];x.XFree.argtypes=[P];x.XCloseDisplay.argtypes=[P]
t.XTestFakeKeyEvent.argtypes=[P,C.c_uint,C.c_int,U]
d=x.XOpenDisplay(None)
if not d: raise SystemExit('Cannot open task display')
import re,subprocess
info=subprocess.check_output(['xwininfo','-root','-tree'],text=True)
windows=re.findall(r'(0x[0-9a-f]+) "[^"\n]* - Chromium"[^\n]*? (\d+)x(\d+)\+',info)
if not windows: raise SystemExit('Task Chromium window not found')
window=int(max(windows,key=lambda w:int(w[1])*int(w[2]))[0],16)
x.XSetInputFocus(d,window,2,0);x.XFlush(d);time.sleep(.1)
def key(name,down): t.XTestFakeKeyEvent(d,x.XKeysymToKeycode(d,x.XStringToKeysym(name.encode())),down,0);x.XFlush(d);time.sleep(.04)
key('Control_L',1);key('0',1);key('0',0);key('Control_L',0);time.sleep(.15)
for _ in range(int(sys.argv[1])):
 key('Control_L',1);key('equal',1);key('equal',0);key('Control_L',0);time.sleep(.1)
x.XCloseDisplay(d)
