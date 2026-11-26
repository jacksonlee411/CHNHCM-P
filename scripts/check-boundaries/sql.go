package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Fold adjacent constant fragments as well as ordinary interpreted/raw literals.
func stringLiteral(n ast.Node) (string, bool) {
	switch v := n.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			s, err := strconv.Unquote(v.Value)
			return s, err == nil
		}
	case *ast.BinaryExpr:
		if v.Op == token.ADD {
			left, ok := stringLiteral(v.X)
			if ok {
				right, ok := stringLiteral(v.Y)
				return left + right, ok
			}
		}
	case *ast.ParenExpr:
		return stringLiteral(v.X)
	}
	return "", false
}

type sqlOwners map[string]bool

func readSQLOwners(path string) (sqlOwners, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var registry struct{ Contexts []struct{ Name string } }
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}
	owners := sqlOwners{}
	for _, entry := range registry.Contexts {
		if entry.Name == "" || owners[entry.Name] {
			return nil, fmt.Errorf("%s: empty or duplicate context name %q", path, entry.Name)
		}
		owners[entry.Name] = true
	}
	if len(owners) == 0 {
		return nil, fmt.Errorf("%s: missing context names", path)
	}
	return owners, nil
}

func sqlOwner(path string) string {
	parts := strings.Split(path, "/")
	switch {
	case len(parts) >= 4 && parts[0] == "db" && parts[1] == "queries" && strings.HasSuffix(path, ".sql"):
	case len(parts) >= 5 && parts[0] == "internal" && parts[1] == "contexts" && parts[3] == "infra" &&
		(strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".sql")) && !strings.HasSuffix(path, "_test.go") && !strings.Contains(path, "/postgres/generated/"):
	default:
		return ""
	}
	return parts[2]
}

// Check explicit relations in the project's SQL literals, including multiline
// and quoted identifiers. Dynamic table construction needs execution-site review.
var sqlToken = regexp.MustCompile(`(?s)--[^\n]*|/\*.*?\*/|'(?:''|[^'])*'|"(?:""|[^"])*"|[a-zA-Z_][a-zA-Z_0-9$]*|[^\s]`)

func (owners sqlOwners) foreignTables(owner, query string) []string {
	if !strings.Contains(query, ".") {
		return nil
	}
	tokens := slices.DeleteFunc(sqlToken.FindAllString(query, -1), func(token string) bool {
		return strings.HasPrefix(token, "--") || strings.HasPrefix(token, "/*")
	})
	for i, token := range tokens {
		if !strings.HasPrefix(token, `"`) {
			tokens[i] = strings.ToLower(token)
		}
	}

	type clause struct {
		inFrom       bool
		functionName string
	}
	clauses := []clause{{}}
	var found []string
	var expectRelation bool
	for i, token := range tokens {
		current := &clauses[len(clauses)-1]
		if token == "(" {
			var name string
			if i > 0 {
				name = tokens[i-1]
			}
			clauses = append(clauses, clause{functionName: name})
			// A joined table can begin inside one or more parentheses. A
			// subquery's SELECT will itself consume the pending relation.
			continue
		}
		if expectRelation && i+2 < len(tokens) && tokens[i+1] == "." {
			schema := strings.Trim(token, `"`)
			if owners[schema] && schema != owner {
				found = append(found, schema+"."+strings.Trim(tokens[i+2], `"`))
			}
		}
		expectRelation = false
		switch token {
		case ")":
			if len(clauses) > 1 {
				clauses = clauses[:len(clauses)-1]
			}
		case "where", "group", "having", "order", "limit", "returning", "union", "set", ";":
			current.inFrom = false
		case "from":
			// FROM can introduce an expression operand instead of a table.
			if i >= 2 && tokens[i-1] == "distinct" && (tokens[i-2] == "is" ||
				i >= 3 && tokens[i-2] == "not" && tokens[i-3] == "is") {
				continue
			}
			switch current.functionName {
			case "extract", "substring", "trim", "overlay":
				continue
			}
			current.inFrom = true
			expectRelation = true
		case "using":
			current.inFrom = true
			expectRelation = true
		case "join", "update", "into", "table", "only", "truncate":
			expectRelation = true
		case ",":
			expectRelation = current.inFrom
		}
	}
	return found
}
