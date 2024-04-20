package container

import (
	"context"
	"fmt"
	"os"

	"github.com/alexellis/go-execute/v2"
	"golang.org/x/term"
)

// EnterInput contains the input parameters for the Enter function
type EnterInput struct {
	// Name of the container to check for existence
	Name string

	// Command to run in the container
	Command string

	// Shell to use in the container
	Shell string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Enter enters a container
func Enter(ctx context.Context, input EnterInput) error {
	command := input.Shell
	if input.Command != "" {
		command = input.Command
	}
	if command == "" {
		command = "/bin/bash"
	}

	args := []string{"container", "exec"}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		args = append(args, "-it")
	}

	args = append(args, input.Name)

	args = append(args, command)
	cmd := execute.ExecTask{
		Command:     "docker",
		Args:        args,
		StreamStdio: true,
		Stdin:       os.Stdin,
	}

	cmd.PrintCommand = input.Trace
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("exec into container failed: %w", err)
	}

	if res.ExitCode != 0 {
		return nil
	}

	return nil
}
