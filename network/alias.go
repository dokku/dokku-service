package network

import (
	"fmt"

	"github.com/gosimple/slug"
)

type AliasInput struct {
	// ServiceName is the name of the container to connect to the network
	ServiceName string

	// ServiceType is the type of service to connect to the network
	ServiceType string
}

// Alias creates a network alias for a container
func Alias(input AliasInput) string {
	return fmt.Sprintf("dokku.%s.%s", input.ServiceType, slug.Make(input.ServiceName))
}
