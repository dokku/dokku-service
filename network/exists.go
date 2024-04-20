package network

import (
	"context"
	"fmt"

	"github.com/alexellis/go-execute/v2"
)

// ExistsInput contains the input parameters for the Exists function
type ExistsInput struct {
	// Name of the network to check for existence
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Exists checks if a network exists
func Exists(ctx context.Context, input ExistsInput) (bool, error) {
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"network", "inspect",
			input.Name,
		},
		StreamStdio: false,
	}

	cmd.PrintCommand = input.Trace
	res, err := cmd.Execute(ctx)
	if err != nil {
		return false, fmt.Errorf("check for network existence failed: %w", err)
	}

	if res.ExitCode != 0 {
		return false, nil
	}

	return true, nil
}
