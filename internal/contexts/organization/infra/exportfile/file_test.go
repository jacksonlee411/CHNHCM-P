package exportfile

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestPrivateFilePublishesOnlySealedBytesAndReleasesReservation(t *testing.T) {
	p := &Pool{Directory: t.TempDir()}
	f, err := p.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	info, err := f.file.Stat()
	if err != nil || info.Mode().Perm() != 0600 || info.Size() != MaxBytes {
		t.Fatal(info, err)
	}
	if entries, err := os.ReadDir(p.Directory); err != nil || len(entries) != 0 {
		t.Fatal("temporary data has a directory entry", entries, err)
	}
	if _, err = f.Read(make([]byte, 1)); err == nil {
		t.Fatal("unsealed output readable")
	}
	data := []byte("完整输出\r\n")
	if _, err = f.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = f.Seal(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(f)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal(got, err)
	}
	if _, err = f.Write([]byte("late")); err == nil {
		t.Fatal("sealed output mutable")
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil || p.active != 0 {
		t.Fatal("close did not release exactly once", err, p.active)
	}
}

func TestFullBudgetCancellationAndTwentyConcurrentReservations(t *testing.T) {
	p := &Pool{Directory: t.TempDir()}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	files := make([]*File, 0, MaxFiles)
	t.Cleanup(func() {
		for _, f := range files {
			_ = f.Close()
		}
	})
	for range MaxFiles {
		f, err := p.Create(ctx)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	var fault *foundation.Fault
	if f, err := p.Create(ctx); f != nil || !errors.As(err, &fault) || fault.Status != 503 || fault.Code != "M02_EXPORT_RESOURCE_LIMIT" {
		t.Fatal("admitted file 21", err)
	}
	f := files[0]
	block := bytes.Repeat([]byte("x"), 1<<20)
	for range 64 {
		if _, err := f.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := f.Write([]byte("x")); n != 0 || !errors.As(err, &fault) || fault.Status != 413 || fault.Code != "M02_EXPORT_SIZE_LIMIT" {
		t.Fatal("accepted byte beyond 64MiB")
	}
	cancel()
	if err := f.Seal(); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel must prevent publishing", err)
	}
	for _, f := range files {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if p.active != 0 {
		t.Fatal(p.active)
	}
	next, err := p.Create(context.Background())
	if err != nil {
		t.Fatal("released slots not reusable", err)
	}
	_ = next.Close()
}
