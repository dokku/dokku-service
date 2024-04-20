package ambassador

import (
	"context"
	"dokku-service/container"
)

type ExistsInput struct {
	// Name is the name of the service
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Exists checks if an ambassador container exists
func Exists(ctx context.Context, input ExistsInput) (bool, error) {
	input.Name = input.Name + ".ambassador"
	return container.Exists(ctx, container.ExistsInput(input))
}
