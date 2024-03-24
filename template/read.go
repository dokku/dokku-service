package template

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:templates
var templates embed.FS

func ReadDockerfile(path string) (io.Reader, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	return bytes.NewReader(b), nil
}

func ReadTemplate(name string) (io.Reader, error) {
	b, err := templates.ReadFile(filepath.Join("templates", name))
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	return bytes.NewReader(b), nil
}

func ReadDir(name string) ([]fs.DirEntry, error) {
	dirEntries, err := templates.ReadDir(filepath.Join("templates", name))
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	return dirEntries, nil
}

func ReadFile(path string) ([]byte, error) {
	return templates.ReadFile(filepath.Join("templates", path))
}
