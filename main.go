package main

import (
	"fmt"
	"os"

	"dokku-service/commands"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "dokku-service"

// Holds the version
var Version string

func main() {
	os.Exit(Run(os.Args[1:]))
}

// Executes the specified subcommand
func Run(args []string) int {
	commandMeta := command.SetupRun(AppName, Version, args)
	commandMeta.Ui = command.HumanZerologUiWithFields(commandMeta.Ui, make(map[string]interface{}, 0))
	c := cli.NewCLI(AppName, Version)
	c.Args = os.Args[1:]
	c.Commands = command.Commands(commandMeta, Commands)
	exitCode, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

// Returns a list of implemented commands
func Commands(meta command.Meta) map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"service-create": func() (cli.Command, error) {
			return &commands.ServiceCreateCommand{Meta: meta}, nil
		},
		"template-info": func() (cli.Command, error) {
			return &commands.TemplateInfoCommand{Meta: meta}, nil
		},
		"template-list": func() (cli.Command, error) {
			return &commands.TemplateListCommand{Meta: meta}, nil
		},
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{Meta: meta}, nil
		},
	}
}
