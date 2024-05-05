package container

import (
	"context"
	"dokku-service/service"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"mvdan.cc/sh/v3/shell"
)

// ExecuteInput contains the input parameters for the Execute function
type ExecuteInput struct {
	// Name of the container to execute a command in
	Name string

	// CommandName is the name of the command in the template to execute
	CommandName string

	// Config specifies the service template config to use
	ConfigOutput service.ConfigOutput

	// StdErrWriter is the writer to write the stderr of the command to
	StdErrWriter io.Writer

	// StdOutWriter is the writer to write the stdout of the command to
	StdOutWriter io.Writer

	// Trace controls whether to print the command being executed
	Trace bool
}

func Execute(ctx context.Context, input ExecuteInput) error {
	if input.CommandName == "" {
		return fmt.Errorf("Invalid command name specified: %s", input.CommandName)
	}

	command, ok := input.ConfigOutput.Template.Commands[input.CommandName]
	if !ok {
		return fmt.Errorf("%s service %s does not support %s command", input.ConfigOutput.Template.Name, input.Name, input.CommandName)
	}

	tmpl, err := template.New("base").Funcs(sprig.FuncMap()).Parse(command)
	if err != nil {
		return fmt.Errorf("failed to parse connect command template: %w", err)
	}

	builder := &strings.Builder{}
	if err := tmpl.Execute(builder, input.ConfigOutput.Config.EnvironmentVariables); err != nil {
		return fmt.Errorf("failed to execute connect command template: %w", err)
	}

	fields, err := shell.Fields(builder.String(), func(key string) string {
		return input.ConfigOutput.Config.EnvironmentVariables[key]
	})
	if err != nil {
		return fmt.Errorf("failed to parse connect command: %w", err)
	}

	return Enter(ctx, EnterInput{
		Name:         input.Name,
		Command:      fields,
		StdErrWriter: input.StdErrWriter,
		StdOutWriter: input.StdOutWriter,
		Trace:        input.Trace,
	})
}
