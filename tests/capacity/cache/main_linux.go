// Command cache evicts only files in a stopped, task-owned PostgreSQL volume.
// It must be launched by scripts/t07-cold-cache.py in a read-only container.
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func resident(fd, size int) (int, error) {
	if size == 0 {
		return 0, nil
	}
	b, e := unix.Mmap(fd, 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if e != nil {
		return 0, e
	}
	defer unix.Munmap(b)
	pages := make([]byte, (size+os.Getpagesize()-1)/os.Getpagesize())
	_, _, errno := unix.Syscall(unix.SYS_MINCORE, uintptr(unsafe.Pointer(&b[0])), uintptr(size), uintptr(unsafe.Pointer(&pages[0])))
	runtime.KeepAlive(b)
	if errno != 0 {
		return 0, errno
	}
	n := 0
	for _, p := range pages {
		if p&1 != 0 {
			n++
		}
	}
	return n, nil
}
func main() {
	const root = "/data/18/docker"
	if os.Getenv("CHNHCM_T07_CACHE") != "1" {
		panic("task-specific runner required")
	}
	if _, e := os.Stat(filepath.Join(root, "postmaster.pid")); !os.IsNotExist(e) {
		panic("PostgreSQL must be cleanly stopped")
	}
	v, e := os.ReadFile(filepath.Join(root, "PG_VERSION"))
	if e != nil || string(v) != "18\n" {
		panic("unexpected task database directory")
	}
	files, bytes, before, after := 0, int64(0), 0, 0
	residual := []map[string]any{}
	rechecked := []map[string]any{}
	e = filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("external database symlink is not allowed")
		}
		if d.IsDir() {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		b, e := resident(int(f.Fd()), int(info.Size()))
		if e != nil {
			return e
		}
		// DONTNEED may leave dirty or writeback pages resident even after a
		// clean PostgreSQL shutdown. Flush this file only, never the host cache.
		if e = unix.Fsync(int(f.Fd())); e != nil {
			return e
		}
		a, first, passes := 0, 0, 0
		for passes < 3 {
			if passes > 0 {
				time.Sleep(10 * time.Millisecond)
			}
			passes++
			if e = unix.Fadvise(int(f.Fd()), 0, 0, unix.FADV_DONTNEED); e != nil {
				return e
			}
			a, e = resident(int(f.Fd()), int(info.Size()))
			if e != nil {
				return e
			}
			if passes == 1 {
				first = a
			}
			if a == 0 {
				break
			}
		}
		if passes > 1 {
			relative, _ := filepath.Rel(root, path)
			rechecked = append(rechecked, map[string]any{"file": relative, "afterFirst": first, "afterFinal": a, "passes": passes})
		}
		if a != 0 {
			relative, _ := filepath.Rel(root, path)
			residual = append(residual, map[string]any{"file": relative, "before": b, "after": a})
		}
		files++
		bytes += info.Size()
		before += b
		after += a
		return nil
	})
	if e != nil {
		panic(e)
	}
	check := map[string]any{"method": "per-file fsync, POSIX_FADV_DONTNEED and mincore, at most three eviction passes; no global drop_caches", "files": files, "bytes": bytes, "residentPagesBefore": before, "residentPagesAfter": after, "residualFiles": residual, "recheckedFiles": rechecked, "pageBytes": os.Getpagesize(), "passed": after == 0}
	_ = json.NewEncoder(os.Stdout).Encode(check)
	if after != 0 {
		os.Exit(1)
	}
}
