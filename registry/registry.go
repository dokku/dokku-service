package registry

import (
	"context"
	"dokku-service/template"
	"os"
)

// Registry represents a collection of service templates
type Registry struct {
	// RegistryPath specifies the path to the registry
	RegistryPath string

	// Templates is a map of service templates
	Templates map[string]template.ServiceTemplate

	// Vendored specifies if the registry is vendored
	Vendored bool
}

// NewRegistryInput represents the input to the NewRegistry function
type NewRegistryInput struct {
	// RegistryPath specifies the path to the registry
	RegistryPath string

	// Vendored specifies if the registry is vendored
	Vendored bool
}

// NewRegistry creates a new Registry
func NewRegistry(ctx context.Context, input NewRegistryInput) (Registry, error) {
	r := Registry{
		RegistryPath: input.RegistryPath,
		Templates:    map[string]template.ServiceTemplate{},
		Vendored:     input.Vendored,
	}
	err := r.Parse(ctx)
	if err != nil {
		return Registry{}, err
	}

	return r, nil
}

// Parse parses the registry
func (r *Registry) Parse(ctx context.Context) error {
	dirEntries, err := os.ReadDir(r.RegistryPath)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		template, err := template.NewServiceTemplate(ctx, template.NewServiceTemplateInput{
			Name:             dirEntry.Name(),
			RegistryPath:     r.RegistryPath,
			VendoredRegistry: r.Vendored,
		})
		if err != nil {
			return err
		}

		r.Templates[template.Name] = template
	}

	return nil
}
