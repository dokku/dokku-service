package container

import (
	"context"
	"dokku-service/logstreamer"
	"fmt"
	"os"
	"sync"

	"github.com/alexellis/go-execute/v2"
)

// DestroyInput contains the input parameters for the Destroy function
type DestroyInput struct {
	// Name of the container to destroy
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Destroy destroys a container
func Destroy(ctx context.Context, input DestroyInput) error {
	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"container", "rm",
			input.Name,
		},
		StreamStdio: false,
		StdOutWriter: logstreamer.NewLogstreamer(logstreamer.NewLogstreamerInput{
			Mutex:  &mu,
			Writer: os.Stdout,
		}),
		StdErrWriter: logstreamer.NewLogstreamer(logstreamer.NewLogstreamerInput{
			Mutex:  &mu,
			Writer: os.Stderr,
		}),
	}

	cmd.PrintCommand = input.Trace
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to destroy container: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}
