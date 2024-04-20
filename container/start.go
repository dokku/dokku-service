package container

import (
	"context"
	"dokku-service/logstreamer"
	"fmt"
	"os"
	"sync"

	"github.com/alexellis/go-execute/v2"
)

// StartInput contains the input parameters for the Start function
type StartInput struct {
	// Name of the container to start
	Name string
}

// Start starts a container
func Start(ctx context.Context, input StartInput) error {
	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:     "docker",
		Args:        []string{"container", "start", input.Name},
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
		return fmt.Errorf("container start for service failed: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}
