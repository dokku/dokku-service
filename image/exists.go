package image

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexellis/go-execute/v2"
)

// ExistsInput contains the input parameters for the Exists function
type ExistsInput struct {
	// Name of the image to check for existence
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Exists checks if a image exists
func Exists(ctx context.Context, input ExistsInput) (bool, error) {
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"image", "inspect",
			input.Name,
		},
		StreamStdio: false,
	}

	if input.Trace {
		fmt.Fprintln(os.Stderr, "exec: ", cmd.Command, strings.Join(cmd.Args, " "))
	}
	res, err := cmd.Execute(ctx)
	if err != nil {
		return false, fmt.Errorf("check for container existence failed: %w", err)
	}

	if res.ExitCode != 0 {
		return false, nil
	}

	return true, nil
}
