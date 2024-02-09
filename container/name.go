package container

import (
	"fmt"
)

type NameInput struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceType is the type of service
	ServiceType string
}

func Name(input NameInput) string {
	return fmt.Sprintf("dokku.%s.%s", input.ServiceType, input.ServiceName)
}
