package container

import (
	"context"
	"fmt"

	"github.com/alexellis/go-execute/v2"
)

type ExistsInput struct {
	Name string
}

func Exists(ctx context.Context, input ExistsInput) (bool, error) {
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"container", "inspect",
			input.Name,
		},
		StreamStdio: false,
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(ctx)
	if err != nil {
		return false, fmt.Errorf("check for service existence failed: %w", err)
	}

	if res.ExitCode != 0 {
		return false, nil
	}

	return true, nil
}
