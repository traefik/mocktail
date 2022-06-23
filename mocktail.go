package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
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
	PkgName    string
	PkgPath    string
	Imports    map[string]struct{}
	Interfaces []InterfaceDesc
}

// InterfaceDesc represent an interface.
type InterfaceDesc struct {
	Name    string
	Methods []*types.Func
}

func main() {
	modulePath, err := getModulePath(os.Getenv("MOCKTAIL_TEST_PATH"))
	if err != nil {
		log.Fatal("get module path", err)
	}

	moduleName, err := getModuleName(modulePath)
	if err != nil {
		log.Fatal("get module name", err)
	}

	root := filepath.Dir(modulePath)

	// moduleName: github.com/traefik/mocktail
	// modulePath: /home/ldez/go/src/github.com/traefik/mocktail/go.mod
	// root: /home/ldez/go/src/github.com/traefik/mocktail

	err = os.Chdir(root)
	if err != nil {
		log.Fatalf("Chdir: %v", err)
	}

	model, err := walk(root, moduleName)
	if err != nil {
		log.Fatalf("walk: %v", err)
	}

	if len(model) == 0 {
		return
	}

	err = generate(model)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}
}

//nolint:gocognit,gocyclo // The complexity is expected.
func walk(root, moduleName string) (map[string]PackageDesc, error) {
	model := make(map[string]PackageDesc)

	importR := importer.ForCompiler(token.NewFileSet(), "source", nil)

	err := filepath.WalkDir(root, func(fp string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == "vendor" {
				return filepath.SkipDir
			}

			return nil
		}

		if d.Name() != srcMockFile {
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

			i := strings.Index(line, commentTagPattern)
			if i <= -1 {
				continue
			}

			interfaceName := line[i+len(commentTagPattern):]

			filePkgName, err := filepath.Rel(root, filepath.Dir(fp))
			if err != nil {
				return err
			}

			var pkgName string
			if index := strings.LastIndex(interfaceName, "."); index > 0 {
				pkgName = interfaceName[:index]
				interfaceName = interfaceName[index+1:]
			} else {
				pkgName = filePkgName
			}

			importPathFile := path.Join(moduleName, filePkgName)
			importPath := path.Join(moduleName, pkgName)

			pkg, err := importR.Import(importPath)
			if err != nil {
				return fmt.Errorf("failed to import %q: %w", importPath, err)
			}

			lookup := pkg.Scope().Lookup(interfaceName)
			if lookup == nil {
				log.Printf("Unable to find: %s", interfaceName)
				continue
			}

			if packageDesc.PkgName == "" {
				packageDesc.PkgName = lookup.Pkg().Name()
				packageDesc.PkgPath = lookup.Pkg().Path()
			}

			interfaceDesc := InterfaceDesc{Name: interfaceName}

			interfaceType := lookup.Type().Underlying().(*types.Interface)

			for i := 0; i < interfaceType.NumMethods(); i++ {
				method := interfaceType.Method(i)

				interfaceDesc.Methods = append(interfaceDesc.Methods, method)

				for _, imp := range getMethodImports(method, importPathFile) {
					packageDesc.Imports[imp] = struct{}{}
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

func getMethodImports(method *types.Func, importPath string) []string {
	signature := method.Type().(*types.Signature)

	var imports []string

	for _, imp := range getTupleImports(signature.Params(), signature.Results()) {
		if imp != "" && imp != importPath {
			imports = append(imports, imp)
		}
	}

	return imports
}

func getTupleImports(tuples ...*types.Tuple) []string {
	var imports []string

	for _, tuple := range tuples {
		for i := 0; i < tuple.Len(); i++ {
			imports = append(imports, getTypeImports(tuple.At(i).Type())...)
		}
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

	case *types.Signature:
		return getTupleImports(v.Params(), v.Results())

	default:
		panic(fmt.Sprintf("OOPS %[1]T %[1]s", t))
	}
}

func generate(model map[string]PackageDesc) error {
	for fp, pkgDesc := range model {
		buffer := bytes.NewBufferString("")

		err := writeImports(buffer, pkgDesc)
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
					PkgPath:       pkgDesc.PkgPath,
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
