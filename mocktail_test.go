package main

import (
	"bytes"
	"fmt"
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

		cmd := exec.Command("go", "test", "-v", "./...")
		cmd.Dir = filepath.Join(testRoot, entry.Name())

		output, err := cmd.CombinedOutput()
		t.Log(string(output))

		require.NoError(t, err)
	}
}

func TestExportable(t *testing.T) {
	interfaceName := "MyTestInterface"
	assertedTpl := `
// myTestInterfaceMock mock of MyTestInterface.
type myTestInterfaceMock struct { mock.Mock }

// %[1]vMyTestInterfaceMock creates a new myTestInterfaceMock.
func %[1]vMyTestInterfaceMock(tb testing.TB) *myTestInterfaceMock {
	tb.Helper()

	m := &myTestInterfaceMock{}
	m.Mock.Test(tb)

	tb.Cleanup(func() { m.AssertExpectations(tb) })

	return m
}
`
	type args struct {
		exported bool
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "private mocks",
			args:    args{exported: false},
			want:    fmt.Sprintf(assertedTpl, "new"),
			wantErr: assert.NoError,
		},
		{
			name:    "public mocks",
			args:    args{exported: true},
			want:    fmt.Sprintf(assertedTpl, "New"),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := bytes.NewBufferString("")
			err := writeMockBase(buffer, interfaceName, tt.args.exported)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, buffer.String())
		})
	}
}
