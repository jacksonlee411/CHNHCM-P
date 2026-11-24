package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// Explicit loopback development diagnostics only. Profiles contain stack/counter
// samples, never HTTP bodies, cookies, DSNs or heap object contents. Normal
// capacity acceptance runs with this option unset.
func localProfiles(insecure bool) func() {
	directory := os.Getenv("CHNHCM_MEASURE_PROFILE_DIR")
	if directory == "" {
		return func() {}
	}
	if !insecure {
		log.Fatal("local profiles require explicit insecure loopback development mode")
	}
	if e := os.MkdirAll(directory, 0700); e != nil {
		log.Fatal("cannot prepare local profile directory")
	}
	stem := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	cpu, e := os.OpenFile(filepath.Join(directory, stem+"-cpu.pprof"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		log.Fatal("cannot create local CPU profile")
	}
	if e = pprof.StartCPUProfile(cpu); e != nil {
		cpu.Close()
		log.Fatal("cannot start local CPU profile")
	}
	return func() {
		pprof.StopCPUProfile()
		cpu.Close()
		allocations, e := os.OpenFile(filepath.Join(directory, stem+"-allocs.pprof"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			log.Print("cannot create local allocation profile")
			return
		}
		defer allocations.Close()
		if e = pprof.Lookup("allocs").WriteTo(allocations, 0); e != nil {
			log.Print("cannot write local allocation profile")
		}
	}
}
