package container

import (
	"context"
	"dokku-service/logstreamer"
	"fmt"
	"os"
	"sync"

	"github.com/alexellis/go-execute/v2"
)

// StopInput contains the input parameters for the Stop function
type StopInput struct {
	// Name of the container to stop
	Name string
}

// Stop stops a container
func Stop(ctx context.Context, input StopInput) error {
	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"container", "stop",
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

	cmd.PrintCommand = true
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}
