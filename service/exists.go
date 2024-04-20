package service

import (
	"context"
	"dokku-service/registry"
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

func Exists(ctx context.Context, input ExistsInput) (bool, error) {
	_, err := Config(ctx, ConfigInput(input))
	return err == nil, err
}
