package image

import "fmt"

type NameInput struct {
	// ServiceName is the name of the container to connect to the network
	ServiceName string

	// ServiceType is the type of service to connect to the network
	ServiceType string
}

func Name(input NameInput) string {
	return fmt.Sprintf("dokku/service-%s:%s", input.ServiceType, input.ServiceName)
}
