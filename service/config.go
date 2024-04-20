package service

import (
	"context"
	"dokku-service/registry"
	"dokku-service/template"
	"encoding/json"
	"fmt"
	"os"
)

// ConfigInput contains the input parameters for the Config function
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

// ConfigOutput contains the output parameters for the Config function
type ConfigOutput struct {
	// Config is the run configuration for the service
	Config RunConfig `json:"config"`

	// Template is the service template
	Template template.ServiceTemplate `json:"template"`
}

// RunConfig represents the configuration for running a service
type RunConfig struct {
	// Arguments are the arguments to pass to the service
	Arguments map[string]string `json:"arguments"`

	// ContainerCreateFlags are the flags to pass to the container create command
	ContainerCreateFlags []string `json:"container_create_flags"`

	// DataRoot is the root data directory
	DataRoot string `json:"data_root"`

	// EnvironmentVariables are the environment variables to pass to the service
	EnvironmentVariables map[string]string `json:"env"`

	// Image is the image to use for the service
	Image RunImageConfig `json:"image"`

	// ImageBuildFlags are the flags to pass to the image build command
	ImageBuildFlags []string `json:"image_build_flags"`

	// ServiceRoot is the root directory for the service
	ServiceRoot string `json:"service_root"`

	// UseVolumes specifies whether to use volumes
	UseVolumes bool `json:"use_volumes"`
}

// RunImageConfig represents the configuration for the image to run
type RunImageConfig struct {
	// Name is the name of the image
	Name string `json:"name"`

	// Tag is the tag of the image
	Tag string `json:"tag"`
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
