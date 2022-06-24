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

const testRoot = "./testdata/src"

func TestMocktail(t *testing.T) {
	if runtime.GOOS == "windows" {
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

		cmd := exec.Command("go", "test", "-v")
		cmd.Dir = filepath.Join(testRoot, entry.Name())

		output, err := cmd.CombinedOutput()
		t.Log(string(output))

		require.NoError(t, err)
	}
}
