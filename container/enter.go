package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alexellis/go-execute/v2"
	"golang.org/x/term"
)

// EnterInput contains the input parameters for the Enter function
type EnterInput struct {
	// Name of the container to enter
	Name string

	// Command to run in the container
	Command []string

	// Shell to use in the container
	Shell string

	// StdErrWriter is the writer to write the stderr of the command to
	StdErrWriter io.Writer

	// StdOutWriter is the writer to write the stdout of the command to
	StdOutWriter io.Writer

	// Trace controls whether to print the command being executed
	Trace bool
}

// Enter enters a container
func Enter(ctx context.Context, input EnterInput) error {
	command := []string{input.Shell}
	if len(input.Command) > 0 {
		command = input.Command
	}
	if len(command) == 0 {
		command = []string{"/bin/bash"}
	}

	args := []string{"container", "exec"}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		args = append(args, "-it")
	}

	args = append(args, input.Name)

	stdoutWriter := io.Writer(os.Stdout)
	if input.StdOutWriter != nil {
		stdoutWriter = input.StdOutWriter
	}
	stderrWriter := io.Writer(os.Stderr)
	if input.StdErrWriter != nil {
		stderrWriter = input.StdErrWriter
	}


	args = append(args, command...)
	cmd := execute.ExecTask{
		Command:      "docker",
		Args:         args,
		StdOutWriter: stdoutWriter,
		StdErrWriter: stderrWriter,
		Stdin:        os.Stdin,
	}

	if input.Trace {
		fmt.Fprintln(os.Stderr, "exec: ", cmd.Command, strings.Join(cmd.Args, " "))
	}
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("exec into container failed: %w", err)
	}

	if res.ExitCode != 0 {
		return nil
	}

	return nil
}
