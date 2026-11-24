package main

import (
	"chnhcm.local/chnhcm/internal/bootstrap/vankeverify"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	mode := flag.String("mode", "size", "initialize, migrate, serve, serve-import, archive, source-audit, dump, maintain, compact, size")
	dir := flag.String("private", "", "external task directory")
	flag.Parse()
	v, e := vankeverify.Run(*mode, *dir)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	if e = json.NewEncoder(os.Stdout).Encode(v); e != nil {
		panic(e)
	}
}
