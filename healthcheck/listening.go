package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexellis/go-execute/v2"
	retry "github.com/avast/retry-go"
	"github.com/docker/docker/api/types"
	"golang.org/x/sync/errgroup"
)

type ListeningCheckInput struct {
	// Attempts is the number of attempts to make
	Attempts int

	// Container is the container to check
	Container types.ContainerJSON

	// InitialNetwork is the network to use for the wait container
	InitialNetwork string

	// NetworkAlias is the network alias to use for the wait container
	NetworkAlias string

	// Ports is the list of ports to check
	Ports []int

	// Timeout is the timeout in seconds
	Timeout int

	// Trace controls whether to print the command being executed
	Trace bool

	// Wait is the time to wait between attempts
	Wait int

	// WaitImage is the image to use for the wait container
	WaitImage string
}

func ListeningCheck(ctx context.Context, input ListeningCheckInput) error {
	if input.Attempts <= 0 {
		input.Attempts = 1
	}
	if input.NetworkAlias == "" {
		return errors.New("missing required network alias input")
	}
	if input.Timeout <= 0 {
		input.Timeout = 5
	}
	if len(input.Ports) == 0 {
		return errors.New("missing required port input")
	}
	if input.Wait <= 0 {
		input.Wait = 1
	}
	if input.WaitImage == "" {
		input.WaitImage = "dokku/wait:0.6.0"
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, port := range input.Ports {
		listenPort := port
		g.Go(func() error {
			return retry.Do(
				func() error {
					return _dockerlisteningCheck(ctx, input, listenPort)
				},
				retry.Context(ctx),
				retry.Attempts(uint(input.Attempts)),
				retry.Delay(time.Duration(input.Wait)*time.Second),
			)
		})
	}

	return g.Wait()
}

func _dockerlisteningCheck(ctx context.Context, input ListeningCheckInput, port int) error {
	if !input.Container.State.Running {
		return errors.New("container state is not running")
	}

	if input.Container.State.Pid == 0 {
		return errors.New("container state is not running")
	}

	args := []string{"container", "run", "--rm", "--link", fmt.Sprintf("%s:%s", input.Container.Name, input.NetworkAlias)}
	if input.InitialNetwork != "" {
		args = append(args, "--network", input.InitialNetwork)
	}

	args = append(args, input.WaitImage)
	args = append(args, "-c", fmt.Sprintf("%s:%d", input.NetworkAlias, port))

	cmd := execute.ExecTask{
		Command:     "docker",
		Args:        args,
		StreamStdio: false,
	}

	cmd.PrintCommand = input.Trace
	result, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("error running dokku/wait on port: %d: %w", port, err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("container is not listening on port: %d", port)
	}

	return nil
}
