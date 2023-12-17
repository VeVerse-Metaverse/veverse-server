package main

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// getConfiguration gets the configuration string from the environment
func getConfiguration() string {
	if pEnvironment == "dev" {
		return "Development"
	} else if pEnvironment == "test" {
		return "Test"
	} else if pEnvironment == "prod" {
		return "Shipping"
	}
	return "Development"
}

// findEntrypoint searches for the possible entrypoint for the server starting with the root directory
func findEntrypoint(root string) (entrypoint string, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, getBinarySuffix()) {
			entrypoint = path
			return nil
		}

		return nil
	})

	if entrypoint == "" {
		return "", fmt.Errorf("no entrypoint found")
	}

	return filepath.Abs(entrypoint)
}

// getProjectName extracts the project name from the entrypoint
func getProjectName(entrypoint string) string {
	// Get the entrypoint file name
	base := path.Base(entrypoint)
	// Get the filename from the entrypoint
	arr := strings.Split(base, "Server")
	return arr[0]
}
