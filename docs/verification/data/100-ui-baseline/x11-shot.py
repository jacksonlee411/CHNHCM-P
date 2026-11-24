import ctypes as C,struct,zlib,sys
class XImage(C.Structure):
 _fields_=[('width',C.c_int),('height',C.c_int),('xoffset',C.c_int),('format',C.c_int),('data',C.c_void_p),('byte_order',C.c_int),('bitmap_unit',C.c_int),('bitmap_bit_order',C.c_int),('bitmap_pad',C.c_int),('depth',C.c_int),('bytes_per_line',C.c_int),('bits_per_pixel',C.c_int)]
x=C.CDLL('libX11.so.6');x.XOpenDisplay.argtypes=[C.c_char_p];x.XOpenDisplay.restype=C.c_void_p;x.XDefaultRootWindow.argtypes=[C.c_void_p];x.XDefaultRootWindow.restype=C.c_ulong
x.XGetImage.argtypes=[C.c_void_p,C.c_ulong,C.c_int,C.c_int,C.c_uint,C.c_uint,C.c_ulong,C.c_int];x.XGetImage.restype=C.POINTER(XImage)
x.XDestroyImage.argtypes=[C.POINTER(XImage)];x.XCloseDisplay.argtypes=[C.c_void_p]
d=x.XOpenDisplay(None);w,h=map(int,sys.argv[2:4]);p=x.XGetImage(d,x.XDefaultRootWindow(d),0,0,w,h,0xffffffff,2);i=p.contents
if i.bits_per_pixel!=32 or i.byte_order!=0: raise SystemExit('Unsupported display format')
b=C.string_at(i.data,i.bytes_per_line*h);rows=[]
for y in range(h):
 row=b[y*i.bytes_per_line:y*i.bytes_per_line+w*4];rgb=bytearray(w*3);rgb[0::3]=row[2::4];rgb[1::3]=row[1::4];rgb[2::3]=row[0::4];rows.append(b'\0'+rgb)
def chunk(t,b):return struct.pack('!I',len(b))+t+b+struct.pack('!I',zlib.crc32(t+b)&0xffffffff)
open(sys.argv[1],'wb').write(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('!2I5B',w,h,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(b''.join(rows)))+chunk(b'IEND',b''))
x.XDestroyImage(p);x.XCloseDisplay(d)
