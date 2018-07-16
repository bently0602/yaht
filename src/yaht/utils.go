package main

import (
	"os"
	"path/filepath"
)

func GetExePath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}