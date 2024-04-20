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

type ConfigOutput struct {
	Config   RunConfig                `json:"config"`
	Template template.ServiceTemplate `json:"template"`
}

type RunConfig struct {
	Arguments            map[string]string `json:"arguments"`
	ContainerCreateFlags []string          `json:"container_create_flags"`
	DataRoot             string            `json:"data_root"`
	EnvironmentVariables map[string]string `json:"env"`
	Image                RunImageConfig    `json:"image"`
	ImageBuildFlags      []string          `json:"image_build_flags"`
	ServiceRoot          string            `json:"service_root"`
	UseVolumes           bool              `json:"use_volumes"`
}

type RunImageConfig struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

func Config(ctx context.Context, input ConfigInput) (ConfigOutput, error) {
	serviceTemplate, ok := input.Registry.ServiceTemplate(ctx, input.ServiceType)
	if !ok {
		return ConfigOutput{}, fmt.Errorf("%s service template not found", input.ServiceType)
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", input.DataRoot, serviceTemplate.Name, input.Name)
	if _, err := os.Stat(serviceRoot); err != nil {
		return ConfigOutput{}, fmt.Errorf("%s service %s not found", input.ServiceType, input.Name)
	}

	configPath := fmt.Sprintf("%s/config.json", serviceRoot)
	if _, err := os.Stat(configPath); err != nil {
		return ConfigOutput{}, fmt.Errorf("%s service %s config not found", input.ServiceType, input.Name)
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return ConfigOutput{}, fmt.Errorf("failed to read service config: %s", err.Error())
	}

	parsedServiceTemplate := ConfigOutput{}
	if err := json.Unmarshal(b, &parsedServiceTemplate); err != nil {
		return ConfigOutput{}, fmt.Errorf("failed to unmarshal service config: %s", err.Error())
	}

	return parsedServiceTemplate, nil
}
