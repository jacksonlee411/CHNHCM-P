// Package exportfile owns request-local, bounded CSV storage. Files are
// unlinked immediately, so a process crash cannot leave business data behind.
package exportfile

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	"chnhcm.local/chnhcm/internal/foundation"
	"golang.org/x/sys/unix"
)

const MaxBytes int64 = 64 << 20
const MaxFiles = 20

// Pool reserves the full per-file capacity before accepting output. Admission
// never waits while holding a business connection, and never truncates CSV.
type Pool struct {
	Directory string
	mu        sync.Mutex
	active    int
}

var Default = &Pool{Directory: "/tmp"}

func unavailable(cause error) error {
	return errors.Join(foundation.Fail(503, "M02_EXPORT_RESOURCE_LIMIT", "导出服务暂时不可用，请稍后重试"), cause)
}

func (p *Pool) Create(ctx context.Context) (*File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active >= MaxFiles {
		return nil, unavailable(nil)
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(p.Directory, &stat); err != nil {
		return nil, unavailable(err)
	}
	// Keep capacity for all twenty concurrent files, including slots not yet
	// occupied; fallocate below makes each accepted reservation physical.
	if stat.Bavail*uint64(stat.Bsize) < uint64(MaxFiles-p.active)*uint64(MaxBytes) {
		return nil, unavailable(nil)
	}
	f, err := os.CreateTemp(p.Directory, "chnhcm-export-*")
	if err != nil {
		return nil, unavailable(err)
	}
	if err = os.Remove(f.Name()); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return nil, unavailable(err)
	}
	if err = unix.Fallocate(int(f.Fd()), 0, 0, MaxBytes); err != nil {
		_ = f.Close()
		return nil, unavailable(err)
	}
	if err = ctx.Err(); err != nil {
		_ = f.Close()
		return nil, err
	}
	p.active++
	return &File{file: f, pool: p, ctx: ctx}, nil
}

type File struct {
	file    *os.File
	pool    *Pool
	ctx     context.Context
	written int64
	sealed  bool
	closed  bool
}

func (f *File) Write(data []byte) (int, error) {
	if err := f.ctx.Err(); err != nil {
		return 0, err
	}
	if f.closed || f.sealed {
		return 0, os.ErrClosed
	}
	if int64(len(data)) > MaxBytes-f.written {
		return 0, foundation.Fail(413, "M02_EXPORT_SIZE_LIMIT", "导出内容过多，请减少所选项或分批导出")
	}
	n, err := f.file.Write(data)
	f.written += int64(n)
	return n, err
}

func (f *File) Seal() error {
	if err := f.ctx.Err(); err != nil {
		return err
	}
	if f.closed || f.sealed {
		return os.ErrClosed
	}
	if err := f.file.Truncate(f.written); err != nil {
		return err
	}
	if _, err := f.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	f.sealed = true
	return nil
}

func (f *File) Read(data []byte) (int, error) {
	if err := f.ctx.Err(); err != nil {
		return 0, err
	}
	if f.closed || !f.sealed {
		return 0, os.ErrClosed
	}
	return f.file.Read(data)
}

func (f *File) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	err := f.file.Close()
	f.pool.mu.Lock()
	f.pool.active--
	f.pool.mu.Unlock()
	return err
}
