// Command storage126 runs the task-owned compact-history verification lane.
package main

import (
	"chnhcm.local/chnhcm/internal/bootstrap/storageverify"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	mode := flag.String("mode", "encode", "prepare, auxiliary, prototypes, extras, background, size, maintain, compact, serve or encode")
	input := flag.String("input", "", "task-owned external input directory or prototype JSON")
	flag.Parse()
	out, err := storageverify.Run(*mode, *input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(out); err != nil {
		panic(err)
	}
}
