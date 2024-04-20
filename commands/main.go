package commands

import (
	"context"
	"dokku-service/registry"
	"dokku-service/template"
	"fmt"
	"os"
)

type TemplateCleanupFunc func() error

func fetchTemplateRegistry(ctx context.Context, registryPath string) (registry.Registry, TemplateCleanupFunc, error) {
	vendoredRegistry := false
	deferredFunction := func() error {
		return nil
	}
	if registryPath == "" {
		dir, err := os.MkdirTemp("", "dokku-service-registry-*")
		if err != nil {
			return registry.Registry{}, deferredFunction, fmt.Errorf("Failed to create temporary directory: %s", err.Error())
		}
		deferredFunction = func() error {
			return os.RemoveAll(dir)
		}

		if _, err := registry.NewVendoredRegistry(ctx, dir); err != nil {
			return registry.Registry{}, deferredFunction, fmt.Errorf("Failed to create vendored registry: %s", err.Error())
		}
		registryPath = dir
		vendoredRegistry = true
	}

	templateRegistry, err := registry.NewRegistry(ctx, registry.NewRegistryInput{
		RegistryPath: registryPath,
		Vendored:     vendoredRegistry,
	})
	if err != nil {
		return registry.Registry{}, deferredFunction, fmt.Errorf("Failed to parse registry: %s", err.Error())
	}

	return templateRegistry, deferredFunction, err
}

func fetchTemplate(templateRegistry registry.Registry, templateName string) (template.ServiceTemplate, error) {
	serviceTemplate, ok := templateRegistry.Templates[templateName]
	if !ok {
		return template.ServiceTemplate{}, fmt.Errorf("Template %s not found", templateName)
	}

	return serviceTemplate, nil
}
