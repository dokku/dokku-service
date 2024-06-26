package network

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexellis/go-execute/v2"
)

// ConnectInput contains the input parameters for the Connect function
type ConnectInput struct {
	// ContainerName is the name of the container to connect to the network
	ContainerName string

	// NetworkAlias is the alias to use for the container on the network
	NetworkAlias string

	// NetworkName is the name of the network to connect to
	NetworkName string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Connect connects a container to a network
func Connect(ctx context.Context, input ConnectInput) error {
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"network", "connect",
			"--alias", input.NetworkAlias,
			input.NetworkName,
			input.ContainerName,
		},
		StreamStdio: false,
	}

	if input.Trace {
		fmt.Fprintln(os.Stderr, "exec: ", cmd.Command, strings.Join(cmd.Args, " "))
	}
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("network connect failed: %w", err)
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("network connect failed: %s", res.Stderr)
	}

	return nil
}
