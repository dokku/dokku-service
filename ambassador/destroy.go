package ambassador

import (
	"context"
	"dokku-service/container"
)

type DestroyInput struct {
	// Name is the name of the service
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Destroy destroys an ambassador container
func Destroy(ctx context.Context, input DestroyInput) error {
	input.Name = input.Name + ".ambassador"
	return container.Destroy(ctx, container.DestroyInput(input))
}
