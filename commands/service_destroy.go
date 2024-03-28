package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/registry"
)

type ServiceDestroyCommand struct {
	command.Meta

	// Context specifies the context to use
	Context context.Context

	// dataRoot specifies the root directory for service data
	dataRoot string

	// registryPath specifies an override path to the registry
	registryPath string

	// useVolumes specifies whether to use volumes or directories for service data
	useVolumes bool
}

func (c *ServiceDestroyCommand) Name() string {
	return "service-destroy"
}

func (c *ServiceDestroyCommand) Synopsis() string {
	return "Service destroy command"
}

func (c *ServiceDestroyCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServiceDestroyCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Does nothing": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *ServiceDestroyCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "template",
		Description: "the template to use",
		Optional:    false,
		Type:        command.ArgumentString,
	})
	args = append(args, command.Argument{
		Name:        "name",
		Description: "the name of the created service",
		Optional:    false,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *ServiceDestroyCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServiceDestroyCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServiceDestroyCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	f.BoolVar(&c.useVolumes, "use-volumes", false, "use volumes instead of a directory on disk for data")
	return f
}

func (c *ServiceDestroyCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServiceDestroyCommand) Run(args []string) int {
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

	templateName := arguments["template"].StringValue()
	serviceTemplate, ok := templateRegistry.Templates[templateName]
	if !ok {
		c.Ui.Error(fmt.Sprintf("Template %s not found", templateName))
		return 1
	}

	serviceName := arguments["name"].StringValue()
	logger.LogHeader1(fmt.Sprintf("Destroying %s service %s", serviceTemplate.Name, serviceName))
	// check if config exists
	// check if container exists

	// if config does not exist but container exists, show an error telling users to manually cleanup

	// destroy container
	// destroy config

	return 0
}
