// Package refactor provides utilities for writing one-off refactoring scripts.
package refactor

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"

	"go4.org/syncutil"

	"github.com/kisielk/gotool"
)

// WalkPkgs parses the packages with import functions given in pkgs, then
// concurrently walks all of them, calling editFn for each node in the resulting
// ASTs. After the walk, it prints the ASTs back out, overwriting the original
// files.
func WalkPkgs(pkgs []string, editFn func(ast.Node)) []error {
	var wg syncutil.Group
	for _, p := range gotool.ImportPaths(pkgs) {
		p := p
		wg.Go(func() error {
			return walkPkg(p, editFn)
		})
	}
	return wg.Errs()
}

func walkPkg(pkgPath string, editFn func(ast.Node)) error {
	printerConf := printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 8,
	}
	pkg, err := build.Import(pkgPath, ".", 0)
	if err != nil {
		return err
	}
	var wg syncutil.Group
	for _, path := range pkg.GoFiles {
		path := filepath.Join(pkg.Dir, path)
		wg.Go(func() error {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(node ast.Node) bool {
				editFn(node)
				return true
			})
			wc, err := os.Create(path)
			if err != nil {
				return err
			}
			defer wc.Close()
			return printerConf.Fprint(wc, fset, f)
		})
	}
	return wg.Err()
}
