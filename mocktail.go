// package main Naive code generator that creates mock implementation using `testify.mock`.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/types"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	srcMockFile            = "mock_test.go"
	outputMockFile         = "mock_gen_test.go"
	outputExportedMockFile = "mock_gen.go"
)

const contextType = "context.Context"

const commentTagPattern = "// mocktail:"

// PackageDesc represent a package.
type PackageDesc struct {
	Pkg        *types.Package
	Imports    map[string]struct{}
	Interfaces []InterfaceDesc
}

// InterfaceDesc represent an interface.
type InterfaceDesc struct {
	Name    string
	Methods []*types.Func
}

func main() {
	info, err := getModuleInfo(os.Getenv("MOCKTAIL_TEST_PATH"))
	if err != nil {
		log.Fatal("get module path", err)
	}

	var exported bool
	var sourceFile string
	var interfaceNames string
	flag.BoolVar(&exported, "e", false, "generate exported mocks")
	flag.StringVar(&sourceFile, "source", "", "source file containing interfaces to mock")
	flag.StringVar(&interfaceNames, "interface", "", "comma-separated list of interface names to mock (used with -source), mock every interface by default (has no effect without -source)")
	flag.Parse()

	root := info.Dir

	err = os.Chdir(root)
	if err != nil {
		log.Fatalf("Chdir: %v", err)
	}

	var model map[string]PackageDesc
	if sourceFile != "" {
		model, err = processSingleFile(sourceFile, root, info.Path, interfaceNames)
		if err != nil {
			log.Fatalf("process single file: %v", err)
		}
	} else {
		model, err = walk(root, info.Path)
		if err != nil {
			log.Fatalf("walk: %v", err)
		}
	}

	if len(model) == 0 {
		return
	}

	err = generate(model, exported)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}
}

// processSingleFile processes a single source file to extract interfaces for mocking.
func processSingleFile(sourceFile, root, moduleName, interfaceFilter string) (map[string]PackageDesc, error) {
	model := make(map[string]PackageDesc)

	// Convert to absolute path if relative
	if !filepath.IsAbs(sourceFile) {
		sourceFile = filepath.Join(os.Getenv("PWD"), sourceFile)
	}

	// Check if file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("source file does not exist: %s", sourceFile)
	}

	// Parse interface filter if provided
	targetInterfaces := parseInterfaceFilter(interfaceFilter)

	// Load package from source file
	pkg, err := loadPackageFromFile(sourceFile, root, moduleName)
	if err != nil {
		return nil, fmt.Errorf("load package from file: %w", err)
	}

	if pkg == nil {
		return model, nil // Return empty model when no packages found
	}

	// Process interfaces in the package
	packageDesc := processPackageInterfaces(pkg, targetInterfaces)

	if len(packageDesc.Interfaces) > 0 {
		// Use the source file path as the key, but change the filename to match expected output location
		outputDir := filepath.Dir(sourceFile)
		outputKey := filepath.Join(outputDir, srcMockFile)
		model[outputKey] = packageDesc
	}

	return model, nil
}

// parseInterfaceFilter parses the interface filter string into a map of target interfaces.
func parseInterfaceFilter(interfaceFilter string) map[string]struct{} {
	if interfaceFilter == "" {
		return nil
	}

	targetInterfaces := make(map[string]struct{})
	for _, name := range strings.Split(interfaceFilter, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			targetInterfaces[name] = struct{}{}
		}
	}

	return targetInterfaces
}

// loadPackageFromFile loads a Go package from a source file.
func loadPackageFromFile(sourceFile, root, moduleName string) (*types.Package, error) {
	// Get the package path for this file
	fileDir := filepath.Dir(sourceFile)

	// Load the package
	pkgs, err := packages.Load(
		&packages.Config{
			Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
			Dir:  fileDir,
		},
		".",
	)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, nil
	}

	pkg := pkgs[0]
	if pkg.Types == nil {
		relDir, err := filepath.Rel(root, fileDir)
		if err != nil {
			return nil, fmt.Errorf("get relative directory: %w", err)
		}

		return nil, fmt.Errorf("package %q has no type information", path.Join(moduleName, relDir))
	}

	return pkg.Types, nil
}

// processPackageInterfaces processes all interfaces in a package, optionally filtering by target interfaces.
func processPackageInterfaces(pkg *types.Package, targetInterfaces map[string]struct{}) PackageDesc {
	packageDesc := PackageDesc{
		Pkg:     pkg,
		Imports: map[string]struct{}{},
	}

	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}

		// If interface filter is specified, only process those interfaces
		if targetInterfaces != nil {
			if _, wanted := targetInterfaces[name]; !wanted {
				continue
			}
		}

		// Check if it's an interface and process it
		interfaceDesc := processInterfaceType(name, obj)

		if interfaceDesc != nil {
			packageDesc.Interfaces = append(packageDesc.Interfaces, *interfaceDesc)
			// Collect imports from the interface methods
			for _, method := range interfaceDesc.Methods {
				for _, imp := range getMethodImports(method, pkg.Path()) {
					packageDesc.Imports[imp] = struct{}{}
				}
			}
		}
	}

	return packageDesc
}

// processInterfaceType processes a single type to check if it's an interface and extract its methods.
func processInterfaceType(name string, obj types.Object) *InterfaceDesc {
	// Check if it's an interface
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}

	interfaceType, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	interfaceDesc := InterfaceDesc{Name: name}

	// Get all methods from the interface
	for i := range interfaceType.NumMethods() {
		method := interfaceType.Method(i)
		interfaceDesc.Methods = append(interfaceDesc.Methods, method)
	}

	if len(interfaceDesc.Methods) == 0 {
		return nil
	}

	return &interfaceDesc
}

//nolint:gocognit,gocyclo // The complexity is expected.
func walk(root, moduleName string) (map[string]PackageDesc, error) {
	model := make(map[string]PackageDesc)

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

			var importPath string
			if index := strings.LastIndex(interfaceName, "."); index > 0 {
				importPath = path.Join(moduleName, interfaceName[:index])

				interfaceName = interfaceName[index+1:]
			} else {
				filePkgName, err := filepath.Rel(root, filepath.Dir(fp))
				if err != nil {
					return err
				}

				importPath = path.Join(moduleName, filePkgName)
			}

			pkgs, err := packages.Load(
				&packages.Config{
					Mode: packages.NeedTypes,
					Dir:  root,
				},
				importPath,
			)
			if err != nil {
				return fmt.Errorf("load package %q: %w", importPath, err)
			}

			// Only one package specified by the import path has been loaded.
			lookup := pkgs[0].Types.Scope().Lookup(interfaceName)
			if lookup == nil {
				log.Printf("Unable to find: %s", interfaceName)
				continue
			}

			if packageDesc.Pkg == nil {
				packageDesc.Pkg = lookup.Pkg()
			}

			interfaceDesc := InterfaceDesc{Name: interfaceName}

			interfaceType, ok := lookup.Type().Underlying().(*types.Interface)
			if !ok {
				return fmt.Errorf("type %q in %q is not an interface", lookup.Type(), fp)
			}

			for i := range interfaceType.NumMethods() {
				method := interfaceType.Method(i)

				interfaceDesc.Methods = append(interfaceDesc.Methods, method)

				for _, imp := range getMethodImports(method, packageDesc.Pkg.Path()) {
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
		for i := range tuple.Len() {
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

	case *types.Array:
		return getTypeImports(v.Elem())

	case *types.Struct:
		var imports []string
		for i := range v.NumFields() {
			imports = append(imports, getTypeImports(v.Field(i).Type())...)
		}
		return imports

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

	case *types.Chan:
		return []string{""}

	default:
		panic(fmt.Sprintf("OOPS %[1]T %[1]s", t))
	}
}

func generate(model map[string]PackageDesc, exported bool) error {
	for fp, pkgDesc := range model {
		buffer := bytes.NewBufferString("")

		err := writeImports(buffer, pkgDesc)
		if err != nil {
			return err
		}

		for _, interfaceDesc := range pkgDesc.Interfaces {
			err = writeMockBase(buffer, interfaceDesc.Name, exported)
			if err != nil {
				return err
			}

			_, _ = buffer.WriteString("\n")

			for _, method := range interfaceDesc.Methods {
				signature := method.Type().(*types.Signature)

				syrup := Syrup{
					PkgPath:       pkgDesc.Pkg.Path(),
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

		fileName := outputMockFile
		if exported {
			fileName = outputExportedMockFile
		}

		out := filepath.Join(filepath.Dir(fp), fileName)

		log.Println(out)

		err = os.WriteFile(out, source, 0o640)
		if err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}
