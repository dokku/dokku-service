package registry

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:templates
var templates embed.FS

// VendoredRegistry represents the internally vendored service template registry
type VendoredRegistry struct{}

// NewVendoredRegistry creates a new VendoredRegistry
func NewVendoredRegistry(ctx context.Context, extractPath string) (VendoredRegistry, error) {
	r := VendoredRegistry{}
	err := r.Extract(ctx, extractPath)
	if err != nil {
		return VendoredRegistry{}, err
	}

	return r, nil
}

// Extract extracts the vendored registry to the specified path
func (r VendoredRegistry) Extract(ctx context.Context, extractPath string) error {
	dirEntries, err := templates.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			err := writeOutDir(ctx, dirEntry, "", extractPath)
			if err != nil {
				return fmt.Errorf("failed to write directory: %w", err)
			}

			continue
		}

		writePath := filepath.Join(extractPath, dirEntry.Name())
		err := writeOutFile(ctx, dirEntry, "", writePath)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// writeOutDir writes a directory to the specified path
func writeOutDir(ctx context.Context, dirEntry fs.DirEntry, basePath string, extractPath string) error {
	if !dirEntry.IsDir() {
		return fmt.Errorf("unexpected file: %s", dirEntry.Name())
	}

	newBasePath := filepath.Join(basePath, dirEntry.Name())
	err := os.MkdirAll(filepath.Join(extractPath, newBasePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dirEntries, err := templates.ReadDir(filepath.Join("templates", newBasePath))
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			err := writeOutDir(ctx, dirEntry, newBasePath, extractPath)
			if err != nil {
				return fmt.Errorf("failed to write directory: %w", err)
			}

			continue
		}

		writePath := filepath.Join(extractPath, newBasePath, dirEntry.Name())
		err := writeOutFile(ctx, dirEntry, newBasePath, writePath)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// writeOutFile writes a file to the specified path
func writeOutFile(_ context.Context, dirEntry fs.DirEntry, basePath string, writePath string) error {
	if dirEntry.IsDir() {
		return fmt.Errorf("cannot write directory: %s", dirEntry.Name())
	}

	contents, err := templates.ReadFile(filepath.Join("templates", basePath, dirEntry.Name()))
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	handle, err := os.Create(writePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = handle.Write(contents)
	if err != nil {
		return fmt.Errorf("failed to write content to file: %w", err)
	}

	// todo: set permissions

	return nil
}
