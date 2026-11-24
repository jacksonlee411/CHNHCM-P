// Command storage130 is an offline, task-owned verification tool.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"chnhcm.local/chnhcm/internal/bootstrap/storagecase130"
)

func main() {
	mode := flag.String("mode", "", "prepare, export, import, verify, size, serve, maintain-dry-run, maintain or compact")
	input := flag.String("input", "", "private directory outside the checkout")
	role := flag.String("role", "", "A baseline or B candidate; only used by prepare")
	flag.Parse()
	value, err := storagecase130.Run(*mode, *input, *role)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(value); err != nil {
		os.Exit(1)
	}
}
