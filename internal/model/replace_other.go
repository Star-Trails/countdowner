//go:build !windows

package model

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
