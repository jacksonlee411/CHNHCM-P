// Command check-boundaries checks Go imports and explicit SQL table ownership.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const module = "chnhcm.local/chnhcm/"

func main() {
	failures, files, sqlFiles := 0, 0, 0
	owners, err := readSQLOwners("docs/architecture/contexts.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".local", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		owner := sqlOwner(filepath.ToSlash(path))
		if owner != "" && !owners[owner] {
			return fmt.Errorf("%s: unregistered SQL owner %s", path, owner)
		}
		checkSQL := func(source, query string) {
			for _, table := range owners.foreignTables(owner, query) {
				fmt.Fprintf(os.Stderr, "%s: %s SQL references %s\n", source, owner, table)
				failures++
			}
		}
		if owner != "" && strings.HasSuffix(path, ".sql") {
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			checkSQL(path, string(source))
			sqlFiles++
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		mode := parser.ImportsOnly
		if owner != "" {
			mode = 0
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, mode)
		if err != nil {
			return err
		}
		if owner != "" {
			ast.Inspect(parsed, func(n ast.Node) bool {
				if query, ok := stringLiteral(n); ok {
					checkSQL(fileSet.Position(n.Pos()).String(), query)
					return false
				}
				return true
			})
			sqlFiles++
		}
		for _, imp := range parsed.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if issue := violation(filepath.ToSlash(path), name); issue != "" {
				fmt.Fprintf(os.Stderr, "%s imports %s: %s\n", path, name, issue)
				failures++
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if failures > 0 {
		os.Exit(1)
	}
	fmt.Printf("Go imports (%d files) and static SQL ownership (%d source files) passed\n", files, sqlFiles)
}

func contextLayer(path string) (string, string) {
	const prefix = "internal/contexts/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func violation(source, imported string) string {
	if imported == "C" {
		return "cgo requires an explicit architecture decision"
	}
	internal := strings.HasPrefix(imported, module)
	target := strings.TrimPrefix(imported, module)
	owner, layer := contextLayer(source)
	other, targetLayer := contextLayer(target)
	standard := !internal && !strings.Contains(strings.Split(imported, "/")[0], ".")
	ioDependency := false
	for _, prefix := range []string{"net", "database", "os", "syscall", "unsafe"} {
		ioDependency = ioDependency || imported == prefix || strings.HasPrefix(imported, prefix+"/")
	}
	foundation := internal && (target == "internal/foundation" || strings.HasPrefix(target, "internal/foundation/"))
	// NFC is a pure, version-pinned text primitive, not an external adapter.
	// Keep this permission limited to the one wrapper and one package.
	unicodePrimitive := source == "internal/foundation/unicode.go" && imported == "golang.org/x/text/unicode/norm"
	if strings.HasPrefix(source, "internal/foundation/") && (ioDependency || (!standard && !foundation && !unicodePrimitive)) {
		return "foundation must not depend on domains, platform or external adapters"
	}
	if strings.HasPrefix(source, "internal/platform/") && other != "" {
		return "platform must depend on ports, not business contexts"
	}
	if owner == "" {
		if other != "" && targetLayer != "public" && !strings.HasPrefix(source, "internal/bootstrap/") {
			return "only the composition root may wire context internals"
		}
		return ""
	}
	if layer == "domain" || layer == "public" {
		if (standard && !ioDependency) || foundation || (internal && other == owner && targetLayer == layer) {
			return ""
		}
		return layer + " may only depend on itself, foundation and the standard library"
	}
	if internal && other != "" && other != owner && targetLayer != "public" {
		return "cross-context dependencies must use public contracts"
	}
	if layer == "application" && (ioDependency || (!standard && !internal)) {
		return "application uses ports for external effects; third-party adapters belong in infra"
	}
	if layer == "application" && internal && ((other == owner && targetLayer == "infra") || strings.HasPrefix(target, "internal/platform") || strings.HasPrefix(target, "internal/bootstrap")) {
		return "application depends on ports; infra is injected by bootstrap"
	}
	return ""
}
