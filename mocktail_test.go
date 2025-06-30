package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goosWindows = "windows"

func TestMocktail(t *testing.T) {
	const testRoot = "./testdata/src"

	if runtime.GOOS == goosWindows {
		t.Skip(runtime.GOOS)
	}

	dir, errR := os.ReadDir(testRoot)
	require.NoError(t, errR)

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}

		t.Setenv("MOCKTAIL_TEST_PATH", filepath.Join(testRoot, entry.Name()))

		output, err := exec.Command("go", "run", ".").CombinedOutput()
		t.Log(string(output))

		require.NoError(t, err)
	}

	errW := filepath.WalkDir(testRoot, func(path string, d fs.DirEntry, errW error) error {
		if errW != nil {
			return errW
		}

		if d.IsDir() || d.Name() != outputMockFile {
			return nil
		}

		genBytes, err := os.ReadFile(path)
		require.NoError(t, err)

		goldenBytes, err := os.ReadFile(path + ".golden")
		require.NoError(t, err)

		assert.Equal(t, string(goldenBytes), string(genBytes))

		return nil
	})
	require.NoError(t, errW)

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}

		cmd := exec.Command("go", "test", "-v", "./...")
		cmd.Dir = filepath.Join(testRoot, entry.Name())

		output, err := cmd.CombinedOutput()
		t.Log(string(output))

		require.NoError(t, err)
	}
}

func TestMocktail_exported(t *testing.T) {
	const testRoot = "./testdata/exported"

	if runtime.GOOS == goosWindows {
		t.Skip(runtime.GOOS)
	}

	dir, errR := os.ReadDir(testRoot)
	require.NoError(t, errR)

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}

		t.Setenv("MOCKTAIL_TEST_PATH", filepath.Join(testRoot, entry.Name()))

		output, err := exec.Command("go", "run", ".", "-e").CombinedOutput()
		t.Log(string(output))

		require.NoError(t, err)
	}

	errW := filepath.WalkDir(testRoot, func(path string, d fs.DirEntry, errW error) error {
		if errW != nil {
			return errW
		}

		if d.IsDir() || d.Name() != outputMockFile {
			return nil
		}

		genBytes, err := os.ReadFile(path)
		require.NoError(t, err)

		goldenBytes, err := os.ReadFile(path + ".golden")
		require.NoError(t, err)

		assert.Equal(t, string(goldenBytes), string(genBytes))

		return nil
	})
	require.NoError(t, errW)

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}

		cmd := exec.Command("go", "test", "-v", "./...")
		cmd.Dir = filepath.Join(testRoot, entry.Name())

		output, err := cmd.CombinedOutput()
		t.Log(string(output))

		require.NoError(t, err)
	}
}

func TestMocktail_source(t *testing.T) {
	const testRoot = "./testdata/source"

	if runtime.GOOS == goosWindows {
		t.Skip(runtime.GOOS)
	}

	testCases := []struct {
		name           string
		expectedOutput string
		extraArgs      []string
	}{
		{
			name:           "a",
			expectedOutput: outputMockFile,
			extraArgs:      nil,
		},
		{
			name:           "b",
			expectedOutput: outputExportedMockFile,
			extraArgs:      []string{"-e"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := filepath.Join(testRoot, tc.name)
			interfacesFile := filepath.Join(testDir, "interfaces.go")

			// Convert to absolute path to avoid path duplication issues
			absTestDir, err := filepath.Abs(testDir)
			require.NoError(t, err)
			absInterfacesFile, err := filepath.Abs(interfacesFile)
			require.NoError(t, err)

			// Set up environment
			t.Setenv("MOCKTAIL_TEST_PATH", absTestDir)

			// Build command args
			args := []string{"run", "."}
			args = append(args, tc.extraArgs...)
			args = append(args, "-source="+absInterfacesFile)

			// Run mocktail with source parameter
			output, err := exec.Command("go", args...).CombinedOutput()
			t.Log(string(output))
			require.NoError(t, err)

			// Check generated file matches golden file
			genPath := filepath.Join(testDir, tc.expectedOutput)
			t.Cleanup(func() {
				_ = os.Remove(genPath)
			})

			goldenPath := genPath + ".golden"

			genBytes, err := os.ReadFile(genPath)
			require.NoError(t, err)

			goldenBytes, err := os.ReadFile(goldenPath)
			require.NoError(t, err)

			assert.Equal(t, string(goldenBytes), string(genBytes))

			cmd := exec.Command("go", "test", "-v", "./...")
			cmd.Dir = testDir

			output, err = cmd.CombinedOutput()
			t.Log(string(output))
			require.NoError(t, err)
		})
	}
}

func TestProcessSingleFile(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == goosWindows {
		t.Skip(runtime.GOOS)
	}

	tests := []struct {
		name         string
		sourceFile   string
		expectedErr  bool
		expectedIntf int // expected number of interfaces
	}{
		{
			name:         "valid_basic_file",
			sourceFile:   "testdata/source/a/interfaces.go",
			expectedErr:  false,
			expectedIntf: 1, // PiniaColada
		},
		{
			name:         "valid_exported_file",
			sourceFile:   "testdata/source/b/interfaces.go",
			expectedErr:  false,
			expectedIntf: 1, // PiniaColada
		},
		{
			name:        "nonexistent_file",
			sourceFile:  "testdata/source/nonexistent.go",
			expectedErr: true,
		},
		{
			name:         "relative_path",
			sourceFile:   "./testdata/source/a/interfaces.go",
			expectedErr:  false,
			expectedIntf: 1, // PiniaColada
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert source file to absolute path to avoid path issues
			absSourceFile, err := filepath.Abs(tt.sourceFile)
			if !tt.expectedErr {
				require.NoError(t, err)
			}

			// Get the module info for the specific test directory
			testDir := filepath.Dir(absSourceFile)
			info, err := getModuleInfo(testDir)
			if !tt.expectedErr {
				require.NoError(t, err)
			}

			// Test processSingleFile function
			model, err := processSingleFile(absSourceFile, info.Dir, info.Path)

			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Should have exactly one entry in the model
			assert.Len(t, model, 1)

			// Check the number of interfaces found
			var totalInterfaces int
			for _, pkgDesc := range model {
				totalInterfaces += len(pkgDesc.Interfaces)
			}
			assert.Equal(t, tt.expectedIntf, totalInterfaces)

			// Verify interfaces have methods
			for _, pkgDesc := range model {
				for _, intf := range pkgDesc.Interfaces {
					assert.NotEmpty(t, intf.Methods, "Interface %s should have methods", intf.Name)
				}
			}
		})
	}
}

func TestProcessSingleFile_InvalidPackage(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == goosWindows {
		t.Skip(runtime.GOOS)
	}

	// Create a temporary file with invalid Go code
	tmpFile, err := os.CreateTemp(t.TempDir(), "invalid_*.go")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	_, err = tmpFile.WriteString("package invalid\n\n// This is not a valid interface\ntype NotAnInterface struct{}\n")
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Use current directory for temporary file test
	cwd, err := os.Getwd()
	require.NoError(t, err)
	info, err := getModuleInfo(cwd)
	require.NoError(t, err)

	// Test processSingleFile with file containing no interfaces
	model, err := processSingleFile(tmpFile.Name(), info.Dir, info.Path)

	require.NoError(t, err)
	assert.Empty(t, model, "Should return empty model when no interfaces found")
}

func TestProcessSingleFile_AbsolutePath(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == goosWindows {
		t.Skip(runtime.GOOS)
	}

	// Test with absolute path
	absPath, err := filepath.Abs("testdata/source/a/interfaces.go")
	require.NoError(t, err)

	// Get module info from the test directory
	testDir := filepath.Dir(absPath)
	info, err := getModuleInfo(testDir)
	require.NoError(t, err)

	model, err := processSingleFile(absPath, info.Dir, info.Path)
	require.NoError(t, err)

	assert.Len(t, model, 1)

	var totalInterfaces int
	for _, pkgDesc := range model {
		totalInterfaces += len(pkgDesc.Interfaces)
	}
	assert.Equal(t, 1, totalInterfaces)
}
