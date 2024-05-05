package container

import (
	"context"
	"dokku-service/service"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/alexellis/go-execute/v2"
	"golang.org/x/term"
	"mvdan.cc/sh/v3/shell"
)

// ConnectInput contains the input parameters for the Connect function
type ConnectInput struct {
	// ContainerName is the name of the container to connect to
	ContainerName string

	// Config specifies the service template config to use
	ConfigOutput service.ConfigOutput

	// Name is the name of the service
	Name string

	// Trace controls whether to print the command being executed
	Trace bool
}

// Connect creates an interactive repl connection to a service
func Connect(ctx context.Context, input ConnectInput) error {
	command, ok := input.ConfigOutput.Template.Commands["connect"]
	if !ok {
		return fmt.Errorf("%s service %s does not support connect command", input.ConfigOutput.Template.Name, input.Name)
	}

	tmpl, err := template.New("base").Funcs(sprig.FuncMap()).Parse(command)
	if err != nil {
		return fmt.Errorf("failed to parse connect command template: %w", err)
	}

	builder := &strings.Builder{}
	if err := tmpl.Execute(builder, input.ConfigOutput.Config.EnvironmentVariables); err != nil {
		return fmt.Errorf("failed to execute connect command template: %w", err)
	}

	args := []string{
		"container",
		"exec",
		"--env=LANG=C.UTF-8",
		"--env=LC_ALL=C.UTF-8",
		"-i",
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		args = append(args, "-t")
	}
	args = append(args, input.ContainerName)

	fields, err := shell.Fields(builder.String(), func(key string) string {
		return input.ConfigOutput.Config.EnvironmentVariables[key]
	})
	if err != nil {
		return fmt.Errorf("failed to parse connect command: %w", err)
	}
	args = append(args, fields...)

	cmd := execute.ExecTask{
		Command:     "docker",
		Args:        args,
		StreamStdio: true,
		Stdin:       os.Stdin,
	}

	if input.Trace {
		fmt.Fprintln(os.Stderr, "exec: ", cmd.Command, strings.Join(cmd.Args, " "))
	}
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute connect command: %w", err)
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("connect command failed: %s", res.Stderr)
	}

	return nil
}
