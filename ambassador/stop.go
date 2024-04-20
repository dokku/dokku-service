package ambassador

import (
	"context"
	"dokku-service/container"
)

type StopInput struct {
	// Name is the name of the service
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Stop stops an ambassador container
func Stop(ctx context.Context, input StopInput) error {
	input.Name = input.Name + ".ambassador"
	return container.Stop(ctx, container.StopInput(input))
}
