package commands

import (
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/registry"
)

type TemplateListCommand struct {
	command.Meta

	// registryPath specifies an override path to the registry
	registryPath string
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
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
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
	logger, ok := c.Ui.(*command.ZerologUi)
	if !ok {
		c.Ui.Error("Unable to fetch logger from cli")
		return 1
	}

	registryPath := c.registryPath
	vendoredRegistry := false
	if c.registryPath == "" {
		dir, err := os.MkdirTemp("", "dokku-service-registry-*")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to create temporary directory: %s", err.Error()))
			return 1
		}
		defer os.RemoveAll(dir)

		if _, err := registry.NewVendoredRegistry(c.Context, dir); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to create vendored registry: %s", err.Error()))
			return 1
		}
		registryPath = dir
		vendoredRegistry = true
	}
	templateRegistry, err := registry.NewRegistry(c.Context, registry.NewRegistryInput{
		RegistryPath: registryPath,
		Vendored:     vendoredRegistry,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to parse registry: %s", err.Error()))
		return 1
	}

	logger.LogHeader1("Templates")
	for _, serviceTemplate := range templateRegistry.Templates {
		c.Ui.Info(fmt.Sprintf("%s: %s", serviceTemplate.Name, serviceTemplate.Description))
	}

	return 0
}
