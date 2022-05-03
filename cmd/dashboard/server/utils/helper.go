package utils

import (
	"os"
	"path"
)

func SelfDir() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return path.Dir(ex), nil
}
