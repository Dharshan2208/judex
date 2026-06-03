package workspace

import (
	"os"
	"path/filepath"
)

func CreateWorkspace() (string, error) {
	dir, err := os.MkdirTemp("temp", "job-*")
	if err != nil {
		return "", err
	}

	return filepath.Abs(dir)
}

func WriteFile(dir string, filename string, content string) (string, error) {
	path := filepath.Join(dir, filename)

	// writes a string content to a specified file path with read and write permissions (0o644)
	err := os.WriteFile(path, []byte(content), 0o644)

	return path, err
}

func Cleanup(dir string) error {
	return os.RemoveAll(dir)
}
