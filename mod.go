package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

type modInfo struct {
	Path      string `json:"Path"`  // module name
	Dir       string `json:"Dir"`   // absolute path to the module
	GoMod     string `json:"GoMod"` // absolute path to the go.mod
	GoVersion string `json:"GoVersion"`
	Main      bool   `json:"Main"`
}

func getModuleInfo(dir string) (modInfo, error) {
	// https://github.com/golang/go/issues/44753#issuecomment-790089020
	cmd := exec.Command("go", "list", "-m", "-json")
	if dir != "" {
		cmd.Dir = dir
	}

	raw, err := cmd.CombinedOutput()
	if err != nil {
		return modInfo{}, fmt.Errorf("command go list: %w: %s", err, string(raw))
	}

	var v modInfo
	err = json.NewDecoder(bytes.NewBuffer(raw)).Decode(&v)
	if err != nil {
		return modInfo{}, fmt.Errorf("unmarshaling error: %w: %s", err, string(raw))
	}

	if v.GoMod == "" {
		return modInfo{}, errors.New("working directory is not part of a module")
	}

	return v, nil
}
