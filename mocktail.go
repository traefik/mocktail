package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	srcMockFile    = "mock_test.go"
	outputMockFile = "mock_gen_test.go"
)

const contextType = "context.Context"

const commentTagPattern = "// mocktail:"

// PackageDesc represent a package.
type PackageDesc struct {
	Imports    map[string]struct{}
	Interfaces []InterfaceDesc
}

// InterfaceDesc represent an interface.
type InterfaceDesc struct {
	Name    string
	Methods []*types.Func
}

func main() {
	modulePath, err := getModulePath()
	if err != nil {
		log.Fatal("get module path", err)
	}

	moduleName, err := getModuleName(modulePath)
	if err != nil {
		log.Fatal("get module name", err)
	}

	model, err := walk(modulePath, moduleName)
	if err != nil {
		log.Fatal("walk", err)
	}

	if len(model) == 0 {
		return
	}

	err = testifyWay(model)
	if err != nil {
		log.Fatal("testifyWay", err)
	}
}

//nolint:gocognit,gocyclo // The complexity is expected.
func walk(modulePath, moduleName string) (map[string]PackageDesc, error) {
	root := filepath.Dir(modulePath)

	model := make(map[string]PackageDesc)

	importR := importer.ForCompiler(token.NewFileSet(), "source", nil)

	err := filepath.WalkDir(root, func(fp string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || d.Name() != srcMockFile {
			return nil
		}

		file, err := os.Open(fp)
		if err != nil {
			return err
		}

		packageDesc := PackageDesc{Imports: map[string]struct{}{}}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			pkgName, err := filepath.Rel(root, filepath.Dir(fp))
			if err != nil {
				return err
			}

			i := strings.Index(line, commentTagPattern)
			if i <= -1 {
				continue
			}

			name := line[i+len(commentTagPattern):]

			importPath := path.Clean(moduleName + "/" + pkgName)

			pkg, err := importR.Import(importPath)
			if err != nil {
				return fmt.Errorf("failed to import %q: %w", importPath, err)
			}

			lookup := pkg.Scope().Lookup(name)
			if lookup == nil {
				log.Printf("Unable to find: %s", name)
				continue
			}

			interfaceDesc := InterfaceDesc{Name: name}

			interfaceType := lookup.Type().Underlying().(*types.Interface)

			for i := 0; i < interfaceType.NumMethods(); i++ {
				method := interfaceType.Method(i)

				interfaceDesc.Methods = append(interfaceDesc.Methods, method)

				signature := method.Type().(*types.Signature)

				imports := getTupleImports(signature.Params())
				imports = append(imports, getTupleImports(signature.Results())...)

				for _, imp := range imports {
					if imp != "" && imp != importPath {
						packageDesc.Imports[imp] = struct{}{}
					}
				}
			}

			packageDesc.Interfaces = append(packageDesc.Interfaces, interfaceDesc)
		}

		if len(packageDesc.Interfaces) > 0 {
			model[fp] = packageDesc
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return model, nil
}

func getTupleImports(tuple *types.Tuple) []string {
	var imports []string

	for i := 0; i < tuple.Len(); i++ {
		imports = append(imports, getTypeImports(tuple.At(i).Type())...)
	}

	return imports
}

func getTypeImports(t types.Type) []string {
	switch v := t.(type) {
	case *types.Basic:
		return []string{""}

	case *types.Slice:
		return getTypeImports(v.Elem())

	case *types.Map:
		imports := getTypeImports(v.Key())
		imports = append(imports, getTypeImports(v.Elem())...)
		return imports

	case *types.Named:
		if v.Obj().Pkg() == nil {
			return []string{""}
		}

		return []string{v.Obj().Pkg().Path()}

	case *types.Pointer:
		return getTypeImports(v.Elem())

	case *types.Interface:
		return []string{""}

	default:
		panic(fmt.Sprintf("OOPS %[1]T %[1]s", t))
	}
}

func testifyWay(model map[string]PackageDesc) error {
	for fp, pkgDesc := range model {
		buffer := bytes.NewBufferString("")

		pkg := filepath.Base(filepath.Dir(fp))

		err := writeImports(buffer, pkg, pkgDesc)
		if err != nil {
			return err
		}

		for _, interfaceDesc := range pkgDesc.Interfaces {
			err = writeMockBase(buffer, interfaceDesc.Name)
			if err != nil {
				return err
			}

			_, _ = buffer.WriteString("\n")

			for _, method := range interfaceDesc.Methods {
				signature := method.Type().(*types.Signature)

				syrup := Syrup{
					PackageName:   pkg,
					InterfaceName: interfaceDesc.Name,
					Method:        method,
					Signature:     signature,
				}

				err = syrup.MockMethod(buffer)
				if err != nil {
					return err
				}

				err = syrup.Call(buffer, interfaceDesc.Methods)
				if err != nil {
					return err
				}
			}
		}

		// gofmt
		source, err := format.Source(buffer.Bytes())
		if err != nil {
			log.Println(buffer.String())
			return fmt.Errorf("source: %w", err)
		}

		out := filepath.Join(filepath.Dir(fp), outputMockFile)

		log.Println(out)

		err = os.WriteFile(out, source, 0o640)
		if err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}

// Writer is a wrapper around Print+ functions.
type Writer struct {
	writer io.Writer
	err    error
}

// Err returns error from the other methods.
func (w *Writer) Err() error {
	return w.err
}

// Print formats using the default formats for its operands and writes to standard output.
func (w *Writer) Print(a ...interface{}) {
	if w.err != nil {
		return
	}

	_, w.err = fmt.Fprint(w.writer, a...)
}

// Printf formats according to a format specifier and writes to standard output.
func (w *Writer) Printf(pattern string, a ...interface{}) {
	if w.err != nil {
		return
	}

	_, w.err = fmt.Fprintf(w.writer, pattern, a...)
}

// Println formats using the default formats for its operands and writes to standard output.
func (w *Writer) Println(a ...interface{}) {
	if w.err != nil {
		return
	}

	_, w.err = fmt.Fprintln(w.writer, a...)
}
