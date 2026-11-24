// Package capacitydiag instruments offline capacity runners only. It is never
// registered on the product server and exposes no profiling HTTP endpoint.
package capacitydiag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

type sqlRowsKey struct{}
type sqlBackendKey struct{}

// Backend identities and wall-clock intervals correlate sampled waits only.
// They never supply wait duration; stage durations remain monotonic.
func RecordSQLWindow(ctx context.Context, pid uint32, started, finished int64) {
	if record, ok := ctx.Value(sqlBackendKey{}).(func(uint32, int64, int64)); ok {
		record(pid, started, finished)
	}
}

// RecordSQLRows is diagnostic-only attribution; counts are never durations.
func RecordSQLRows(ctx context.Context, name string, rows int64) {
	if record, ok := ctx.Value(sqlRowsKey{}).(func(string, int64)); ok {
		record(name, rows)
	}
}

// Open is disabled unless an external, private output directory is explicit.
// Profile kinds use separate runs to keep sampling overhead distinguishable.
func Open(next http.Handler) (http.Handler, func(), error) {
	dir := os.Getenv("CHNHCM_CAPACITY_DIAGNOSTICS")
	if dir == "" {
		return next, func() {}, nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	rel, err := filepath.Rel(cwd, abs)
	info, statErr := os.Stat(abs)
	if err != nil || statErr != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, nil, fmt.Errorf("capacity diagnostics require an external 0700 directory")
	}
	open := func(name string) (*os.File, error) {
		return os.OpenFile(filepath.Join(abs, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	}
	log, err := open("requests.jsonl")
	if err != nil {
		return nil, nil, err
	}
	profile := os.Getenv("CHNHCM_CAPACITY_PROFILE")
	var cpu *os.File
	switch profile {
	case "":
	case "cpu":
		cpu, err = open("cpu.pprof")
		if err == nil {
			err = pprof.StartCPUProfile(cpu)
		}
	case "allocs":
		runtime.MemProfileRate = 64 << 10
	case "mutex":
		runtime.SetMutexProfileFraction(1)
	case "block":
		runtime.SetBlockProfileRate(1)
	default:
		err = fmt.Errorf("unsupported capacity profile")
	}
	if err != nil {
		log.Close()
		if cpu != nil {
			cpu.Close()
		}
		return nil, nil, err
	}
	var writes sync.Mutex
	var sequence atomic.Uint64
	var active atomic.Int64
	epoch := time.Now()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, id := time.Now(), sequence.Add(1)
		concurrent := active.Add(1) > 1
		defer active.Add(-1)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		type stage struct {
			Name                string
			StartMS, DurationMS float64
		}
		stages := []stage{}
		sqlRows := map[string]int64{}
		backends := map[uint32]bool{}
		type sqlWindow struct {
			PID                           uint32
			StartedUnixNS, FinishedUnixNS int64
		}
		windows := []sqlWindow{}
		var mu sync.Mutex
		ctx := foundation.WithStageObserver(r.Context(), func(name string, elapsed time.Duration) {
			end := time.Since(start)
			mu.Lock()
			stages = append(stages, stage{name, float64(end-elapsed) / 1e6, float64(elapsed) / 1e6})
			mu.Unlock()
		})
		ctx = context.WithValue(ctx, sqlRowsKey{}, func(name string, rows int64) {
			mu.Lock()
			sqlRows[name] += rows
			mu.Unlock()
		})
		ctx = context.WithValue(ctx, sqlBackendKey{}, func(pid uint32, started, finished int64) {
			mu.Lock()
			backends[pid] = true
			windows = append(windows, sqlWindow{pid, started, finished})
			mu.Unlock()
		})
		w.Header().Set("X-Capacity-Request", fmt.Sprint(id))
		next.ServeHTTP(w, r.WithContext(ctx))
		runtime.ReadMemStats(&after)
		mu.Lock()
		defer mu.Unlock()
		writes.Lock()
		defer writes.Unlock()
		end := time.Now()
		elapsed := end.Sub(start)
		wallDelta := time.Duration(end.UnixNano()-start.UnixNano()) - elapsed
		pids := make([]uint32, 0, len(backends))
		for pid := range backends {
			pids = append(pids, pid)
		}
		// No URLs, headers, cookies, subjects, inputs or business values.
		_ = json.NewEncoder(log).Encode(struct {
			Request                uint64
			Method                 string
			StartedAfterOpenMS     float64
			DurationMS             float64
			Stages                 []stage
			SQLRows                map[string]int64
			ProcessAllocatedBytes  uint64
			ProcessHeapChangeBytes int64
			ProcessMallocs         uint64
			ProcessGCCycles        uint32
			ProcessGCPauseNS       uint64
			ConcurrentMemoryWindow bool
			BackendPIDs            []uint32
			StartedUnixNS          int64
			FinishedUnixNS         int64
			WallClockConsistent    bool
			SQLWindows             []sqlWindow
		}{id, r.Method, float64(start.Sub(epoch)) / 1e6, float64(elapsed) / 1e6, stages, sqlRows,
			after.TotalAlloc - before.TotalAlloc, int64(after.HeapAlloc) - int64(before.HeapAlloc),
			after.Mallocs - before.Mallocs, after.NumGC - before.NumGC, after.PauseTotalNs - before.PauseTotalNs, concurrent || sequence.Load() != id,
			pids, start.UnixNano(), end.UnixNano(), wallDelta > -time.Millisecond && wallDelta < time.Millisecond, windows})
	})
	return h, func() {
		if cpu != nil {
			pprof.StopCPUProfile()
			cpu.Close()
		}
		if profile != "" && profile != "cpu" {
			if profile == "allocs" {
				runtime.GC()
			}
			if f, e := open(profile + ".pprof"); e == nil {
				_ = pprof.Lookup(profile).WriteTo(f, 0)
				f.Close()
			}
		}
		log.Close()
	}, nil
}
