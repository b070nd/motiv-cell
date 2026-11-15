package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func projectPrefix() string {
	return "github.com/b070nd/motiv-cell/src/"
}

func universalStd() []string {
	return []string{"bufio", "bytes", "cmp", "errors", "fmt", "io", "math/bits", "slices", "sort", "strconv", "strings", "unicode", "unicode/utf8"}
}

func extraStd(pkg string) []string {
	switch pkg {
	case "cmd/motivation":
		return []string{"os", "flag"}
	case "reading":
		return []string{"crypto/sha256", "encoding/hex"}
	case "evidence":
		return []string{"encoding/json"}
	case "newsfeed":
		return []string{"encoding/json", "time"}
	case "annotate":
		return []string{"encoding/json"}
	case "laya":
		return []string{"context", "encoding/json", "net/http", "net/url"}
	}
	return nil
}

func projectAllowed() map[string][]string {
	return map[string][]string{
		"model":     {},
		"canon":     {},
		"cell":      {},
		"segment":   {},
		"lattice":   {"cell"},
		"engine":    {"cell", "lattice"},
		"motive":    {"model", "cell"},
		"newsfeed":  {"model"},
		"reading":   {"model", "segment"},
		"annotate":  {"canon"},
		"laya":      {"annotate"},
		"evidence":  {"annotate"},
		"narrative": {"model", "cell", "lattice", "engine", "motive", "reading", "annotate", "evidence"},
		"report":    {"narrative", "canon"},
	}
}

func isCmdPkg(pkg string) bool {
	return pkg == "cmd/motivation"
}

func knownPackage(pkg string) bool {
	if isCmdPkg(pkg) {
		return true
	}
	_, ok := projectAllowed()[pkg]
	return ok
}

func strictPackage(pkg string) bool {
	switch pkg {
	case "cell", "lattice", "engine", "canon":
		return true
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func importAllowed(pkg, path string) bool {
	if contains(universalStd(), path) || contains(extraStd(pkg), path) {
		return true
	}
	if strings.HasPrefix(path, projectPrefix()) {
		rest := strings.TrimPrefix(path, projectPrefix())
		if isCmdPkg(pkg) {
			return rest != ""
		}
		return contains(projectAllowed()[pkg], rest)
	}
	return false
}

func wallClockNames() []string {
	return []string{"Now", "Since", "Until", "Sleep", "After", "AfterFunc", "Tick", "NewTimer", "NewTicker"}
}

func bannedIdents() []string {
	return []string{"float32", "float64", "complex64", "complex128"}
}

func violation(fset *token.FileSet, pos token.Pos, rule, msg string) string {
	p := fset.Position(pos)
	return fmt.Sprintf("%s:%d: %s: %s", filepath.ToSlash(p.Filename), p.Line, rule, msg)
}

func checkFile(f *ast.File, fset *token.FileSet, pkg string) []string {
	var out []string
	timeName := ""
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			out = append(out, violation(fset, spec.Pos(), "R1", "unparsable import path "+spec.Path.Value))
			continue
		}
		if spec.Name != nil && spec.Name.Name == "_" {
			out = append(out, violation(fset, spec.Pos(), "R1", "blank import of "+path))
			continue
		}
		if spec.Name != nil && spec.Name.Name == "." {
			out = append(out, violation(fset, spec.Pos(), "R1", "dot import of "+path))
			continue
		}
		if !importAllowed(pkg, path) {
			out = append(out, violation(fset, spec.Pos(), "R1", fmt.Sprintf("import %q not allowed in package %s", path, pkg)))
		}
		if path == "time" {
			timeName = "time"
			if spec.Name != nil {
				timeName = spec.Name.Name
			}
		}
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "init" {
				out = append(out, violation(fset, d.Pos(), "R2", "func init is forbidden"))
			}
		case *ast.GenDecl:
			if d.Tok == token.VAR {
				out = append(out, violation(fset, d.Pos(), "R3", "package-level var is forbidden"))
			}
		}
	}
	strict := strictPackage(pkg)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if timeName != "" {
				if id, ok := x.X.(*ast.Ident); ok && id.Name == timeName && contains(wallClockNames(), x.Sel.Name) {
					out = append(out, violation(fset, x.Pos(), "R4", "wall clock use time."+x.Sel.Name))
				}
			}
		case *ast.Ident:
			if strict && contains(bannedIdents(), x.Name) {
				out = append(out, violation(fset, x.Pos(), "R5", "identifier "+x.Name+" forbidden in strict package "+pkg))
			}
		case *ast.MapType:
			if strict {
				out = append(out, violation(fset, x.Pos(), "R5", "map type forbidden in strict package "+pkg))
			}
		case *ast.ChanType:
			if strict {
				out = append(out, violation(fset, x.Pos(), "R5", "channel type forbidden in strict package "+pkg))
			}
		case *ast.GoStmt:
			if strict {
				out = append(out, violation(fset, x.Pos(), "R5", "go statement forbidden in strict package "+pkg))
			}
		case *ast.SelectStmt:
			if strict {
				out = append(out, violation(fset, x.Pos(), "R5", "select statement forbidden in strict package "+pkg))
			}
		}
		return true
	})
	return out
}

func collectPackages(t *testing.T, root string) map[string][]string {
	t.Helper()
	pkgs := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "guard" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		pkgs[dir] = append(pkgs[dir], path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return pkgs
}

func TestGuard(t *testing.T) {
	pkgs := collectPackages(t, "..")
	keys := make([]string, 0, len(pkgs))
	for k := range pkgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		files := pkgs[key]
		sort.Strings(files)
		if key == "cmd" {
			continue
		}
		if !knownPackage(key) {
			t.Errorf("%s:1: R1: unknown package %q (register it in the guard tables)", filepath.ToSlash(files[0]), key)
			continue
		}
		for _, file := range files {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Errorf("%s:1: parse: %v", filepath.ToSlash(file), err)
				continue
			}
			for _, v := range checkFile(f, fset, key) {
				t.Error(v)
			}
		}
	}
}

func hasRule(vs []string, rule string) bool {
	for _, v := range vs {
		if strings.Contains(v, ": "+rule+": ") {
			return true
		}
	}
	return false
}

func checkSource(t *testing.T, pkg, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "snippet.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse snippet: %v", err)
	}
	return checkFile(f, fset, pkg)
}

func TestGuardDetects(t *testing.T) {
	cases := []struct {
		name string
		rule string
		pkg  string
		src  string
	}{
		{"R1 foreign std", "R1", "cell", "package p\nimport \"os\"\nvar _ = os.Args\n"},
		{"R1 project", "R1", "cell", "package p\nimport _ \"github.com/b070nd/motiv-cell/src/model\"\n"},
		{"R1 project not allowed", "R1", "lattice", "package p\nimport \"github.com/b070nd/motiv-cell/src/engine\"\n"},
		{"R1 unsafe", "R1", "model", "package p\nimport \"unsafe\"\n"},
		{"R1 dot", "R1", "model", "package p\nimport . \"fmt\"\n"},
		{"R1 cgo", "R1", "model", "package p\nimport \"C\"\n"},
		{"R2", "R2", "model", "package p\nfunc init() {}\n"},
		{"R3", "R3", "model", "package p\nvar x = 1\n"},
		{"R3 grouped", "R3", "model", "package p\nvar (\n\ta = 1\n)\n"},
		{"R4", "R4", "model", "package p\nimport \"time\"\nfunc f() { _ = time.Now() }\n"},
		{"R4 renamed", "R4", "model", "package p\nimport t \"time\"\nfunc f() { t.Sleep(1) }\n"},
		{"R5 float", "R5", "cell", "package p\nfunc f() float64 { return 0 }\n"},
		{"R5 complex", "R5", "engine", "package p\nfunc f(c complex128) {}\n"},
		{"R5 map", "R5", "lattice", "package p\ntype M map[int]int\n"},
		{"R5 chan", "R5", "cell", "package p\ntype C chan int\n"},
		{"R5 go", "R5", "canon", "package p\nfunc f() { go f() }\n"},
		{"R5 select", "R5", "cell", "package p\nfunc f() { select {} }\n"},
	}
	for _, c := range cases {
		vs := checkSource(t, c.pkg, c.src)
		if !hasRule(vs, c.rule) {
			t.Errorf("%s: expected %s violation, got %v", c.name, c.rule, vs)
		}
	}

	clean := []struct {
		name string
		pkg  string
		src  string
	}{
		{"strict", "engine", "package p\nimport (\n\t\"fmt\"\n\t\"github.com/b070nd/motiv-cell/src/cell\"\n)\nconst k = 1\ntype T struct{ a int }\nfunc (T) init() {}\nfunc f() string { return fmt.Sprint(k) }\nfunc g() { _ = cell.X }\n"},
		{"loose", "model", "package p\nimport \"strings\"\ntype M map[string]float64\nfunc f() { go func() {}(); _ = strings.ToLower(\"A\") }\n"},
		{"cmd", "cmd/motivation", "package p\nimport (\n\t\"flag\"\n\t\"os\"\n\t\"github.com/b070nd/motiv-cell/src/store\"\n)\nfunc f() { _ = flag.Args; _ = os.Args; _ = store.X }\n"},
	}
	for _, c := range clean {
		vs := checkSource(t, c.pkg, c.src)
		if len(vs) != 0 {
			t.Errorf("%s: expected no violations, got %v", c.name, vs)
		}
	}
}
