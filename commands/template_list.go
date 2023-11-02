package commands

import (
	"dokku-service/template"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type TemplateListCommand struct {
	command.Meta
}

func (c *TemplateListCommand) Name() string {
	return "template-list"
}

func (c *TemplateListCommand) Synopsis() string {
	return "template-list command"
}

func (c *TemplateListCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *TemplateListCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Lists all templates": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *TemplateListCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	return args
}

func (c *TemplateListCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *TemplateListCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *TemplateListCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	return f
}

func (c *TemplateListCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *TemplateListCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	_, err := c.ParsedArguments(flags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	dirEntries, err := os.ReadDir("templates")
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	logger, ok := c.Ui.(*command.ZerologUi)
	if !ok {
		c.Ui.Error("Unable to fetch logger from cli")
		return 1
	}

	logger.LogHeader1("Templates")
	for _, dirEntry := range dirEntries {
		entry, err := template.ParseDockerfile(fmt.Sprintf("templates/%s", dirEntry.Name()))
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Template parse failure: %s", err.Error()))
			continue
		}
		c.Ui.Info(fmt.Sprintf("%s: %s", entry.Name, entry.Description))
	}

	return 0
}
