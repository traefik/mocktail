package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/mod/modfile"
)

type modInfo struct {
	Path      string `json:"Path"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
	Main      bool   `json:"Main"`
}

func getModulePath(dir string) (string, error) {
	// https://github.com/golang/go/issues/44753#issuecomment-790089020
	cmd := exec.Command("go", "list", "-m", "-json")
	if dir != "" {
		cmd.Dir = dir
	}

	raw, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command go list: %w: %s", err, string(raw))
	}

	var v modInfo
	err = json.NewDecoder(bytes.NewBuffer(raw)).Decode(&v)
	if err != nil {
		return "", fmt.Errorf("unmarshaling error: %w: %s", err, string(raw))
	}

	if v.GoMod == "" {
		return "", errors.New("working directory is not part of a module")
	}

	return v.GoMod, nil
}

func getModuleName(modulePath string) (string, error) {
	raw, err := os.ReadFile(modulePath)
	if err != nil {
		return "", fmt.Errorf("reading go.mod file: %w", err)
	}

	modData, err := modfile.Parse("go.mod", raw, nil)
	if err != nil {
		return "", fmt.Errorf("parsing go.mod file: %w", err)
	}

	return modData.Module.Mod.String(), nil
}
