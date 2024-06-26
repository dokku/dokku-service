package container

import (
	"context"
	"dokku-service/logstreamer"
	"dokku-service/volume"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/alexellis/go-execute/v2"
	"mvdan.cc/sh/v3/shell"
)

type CreateInput struct {
	// CreateFlags specifies the flags to use when creating the container
	CreateFlags []string

	// ContainerName specifies the name of the container to create
	ContainerName string

	// EnvFile specifies the file to use for environment variables
	EnvFile string

	// ImageName specifies the name of the image to use
	ImageName string

	// ServiceRoot specifies the root directory for the service
	ServiceRoot string

	// Trace controls whether to print the command being executed
	Trace bool

	// UseVolumes specifies whether to use volumes
	UseVolumes bool

	// Volumes specifies the volumes to use
	Volumes []volume.Volume
}

// Create creates a container
func Create(ctx context.Context, input CreateInput) error {
	cmdArgs := []string{
		"container", "create",
		"--name", input.ContainerName,
		"--env-file", input.EnvFile,
		"--restart", "always",
		"--hostname", input.ContainerName,
		"--cidfile", fmt.Sprintf("%s/ID", input.ServiceRoot),
	}
	cmdArgs = append(cmdArgs, "--label", fmt.Sprintf("com.dokku.service-volumes=%s", strconv.FormatBool(input.UseVolumes)))

	for _, createFlag := range input.CreateFlags {
		out, _ := shell.Fields(createFlag, nil)
		cmdArgs = append(cmdArgs, out...)
	}

	for _, volume := range input.Volumes {
		cmdArgs = append(cmdArgs, "--mount", volume.MountArgs)
	}

	cmdArgs = append(cmdArgs, input.ImageName)

	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:     "docker",
		Args:        cmdArgs,
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

	if input.Trace {
		fmt.Fprintln(os.Stderr, "exec: ", cmd.Command, strings.Join(cmd.Args, " "))
	}
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("container create for service failed: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}
