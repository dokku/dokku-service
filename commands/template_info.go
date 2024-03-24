package commands

import (
	"dokku-service/template"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type TemplateInfoCommand struct {
	command.Meta
}

func (c *TemplateInfoCommand) Name() string {
	return "template-info"
}

func (c *TemplateInfoCommand) Synopsis() string {
	return "template-info command"
}

func (c *TemplateInfoCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *TemplateInfoCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Show info about template": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *TemplateInfoCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "template",
		Description: "the template to use",
		Optional:    false,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *TemplateInfoCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *TemplateInfoCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *TemplateInfoCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	return f
}

func (c *TemplateInfoCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *TemplateInfoCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	arguments, err := c.ParsedArguments(flags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	logger, ok := c.Ui.(*command.ZerologUi)
	if !ok {
		c.Ui.Error("Unable to fetch logger from cli")
		return 1
	}

	templateName := arguments["template"].StringValue()
	logger.LogHeader1(fmt.Sprintf("%s info", templateName))
	entry, err := template.ParseDockerfile(fmt.Sprintf("templates/%s", templateName))
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Template parse failure: %s", err.Error()))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("name: %s", entry.Name))
	c.Ui.Info(fmt.Sprintf("description: %s", entry.Description))
	c.Ui.Info("arguments:")
	for _, argument := range entry.Arguments {
		defaultValue := ""
		isRequired := false
		if argument.Value == "" {
			defaultValue = "none"
			isRequired = true
		}
		if argument.IsVariable {
			defaultValue = "generated on create"
		} else if argument.Value != "" {
			defaultValue = fmt.Sprintf(`"%s"`, argument.Value)
		}

		c.Ui.Info(fmt.Sprintf("- %s [default: %v, required: %v]", argument.Name, defaultValue, isRequired))
	}

	return 0
}
