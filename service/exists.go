package service

import (
	"context"
	"dokku-service/registry"
	"fmt"
	"os"
)

type ExistsInput struct {
	// DataRoot is the root data directory
	DataRoot string

	// The name of the service
	Name string

	// Registry is the registry to use
	Registry registry.Registry

	// ServiceType is the type of service
	ServiceType string
}

func Exists(ctx context.Context, input ExistsInput) bool {
	serviceTemplate, ok := input.Registry.ServiceTemplate(ctx, input.ServiceType)
	if !ok {
		return false
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", input.DataRoot, serviceTemplate.Name, input.Name)
	if _, err := os.Stat(serviceRoot); err != nil {
		return false
	}

	configPath := fmt.Sprintf("%s/config.json", serviceRoot)
	if _, err := os.Stat(configPath); err != nil {
		return false
	}

	return true
}
