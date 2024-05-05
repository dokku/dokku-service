package container

import (
	"context"
	"dokku-service/logstreamer"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alexellis/go-execute/v2"
)

// LogsInput contains the input parameters for the Logs function
type LogsInput struct {
	// Follow specifies whether to follow the logs
	Follow bool

	// Name of the container to get logs from
	Name string

	// Tail is the number of lines to show from the end of the logs
	Tail int

	// Trace controls whether to print the command being executed
	Trace bool
}

// Logs gets logs from a container
func Logs(ctx context.Context, input LogsInput) error {
	args := []string{"container", "logs"}
	if input.Follow {
		args = append(args, "--follow")
	}
	if input.Tail < 0 {
		args = append(args, "--tail", "all")
	} else if input.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", input.Tail))
	}
	args = append(args, input.Name)

	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:     "docker",
		Args:        args,
		StreamStdio: false,
		StdOutWriter: logstreamer.NewLogstreamer(logstreamer.NewLogstreamerInput{
			DisablePrefix: true,
			Mutex:         &mu,
			Writer:        os.Stdout,
		}),
		StdErrWriter: logstreamer.NewLogstreamer(logstreamer.NewLogstreamerInput{
			DisablePrefix: true,
			Mutex:         &mu,
			Writer:        os.Stderr,
		}),
	}

	if input.Trace {
		fmt.Fprintln(os.Stderr, "exec: ", cmd.Command, strings.Join(cmd.Args, " "))
	}
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
