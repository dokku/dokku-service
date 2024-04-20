package service

import (
	"context"
	"dokku-service/registry"
	"dokku-service/template"
	"encoding/json"
	"fmt"
	"os"
)

type ConfigInput struct {
	// DataRoot is the root data directory
	DataRoot string

	// The name of the service
	Name string

	// Registry is the registry to use
	Registry registry.Registry

	// ServiceType is the type of service
	ServiceType string
}

func Config(ctx context.Context, input ConfigInput) (template.ServiceTemplate, error) {
	serviceTemplate, ok := input.Registry.ServiceTemplate(ctx, input.ServiceType)
	if !ok {
		return template.ServiceTemplate{}, fmt.Errorf("%s service template not found", input.ServiceType)
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", input.DataRoot, serviceTemplate.Name, input.Name)
	if _, err := os.Stat(serviceRoot); err != nil {
		return template.ServiceTemplate{}, fmt.Errorf("%s service %s not found", input.ServiceType, input.Name)
	}

	configPath := fmt.Sprintf("%s/config.json", serviceRoot)
	if _, err := os.Stat(configPath); err != nil {
		return template.ServiceTemplate{}, fmt.Errorf("%s service %s config not found", input.ServiceType, input.Name)
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return template.ServiceTemplate{}, fmt.Errorf("failed to read service config: %s", err.Error())
	}

	parsedServiceTemplate := template.ServiceTemplate{}
	if err := json.Unmarshal(b, &parsedServiceTemplate); err != nil {
		return template.ServiceTemplate{}, fmt.Errorf("failed to unmarshal service config: %s", err.Error())
	}

	return parsedServiceTemplate, nil
}
