package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/container"
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

	templateRegistry, defferedTemplateFunc, err := templateRegistry(c.Context, c.registryPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	defer defferedTemplateFunc()

	templateName := arguments["template"].StringValue()
	serviceTemplate, ok := templateRegistry.Templates[templateName]
	if !ok {
		c.Ui.Error(fmt.Sprintf("Template %s not found", templateName))
		return 1
	}

	serviceName := arguments["name"].StringValue()
	logger.LogHeader1(fmt.Sprintf("Destroying %s service %s", serviceTemplate.Name, serviceName))

	containerName := container.Name(container.NameInput{
		ServiceName: serviceName,
		ServiceType: serviceTemplate.Name,
	})
	containerExists, err := container.Exists(c.Context, container.ExistsInput{
		Name: containerName,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for container existence: %s", err.Error()))
		return 1
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", c.dataRoot, serviceTemplate.Name, serviceName)
	if _, err := os.Stat(serviceRoot); err != nil {
		// todo: handle deleting the service container
		c.Ui.Error(fmt.Sprintf("Failed to check for service data existence: %s", err.Error()))
		if containerExists {
			c.Ui.Error(fmt.Sprintf("Please manually cleanup the service container: %s", containerName))
		}

		return 1
	}

	var destroyErr error
	if containerExists {
		stopErr := container.Stop(c.Context, container.StopInput{
			Name: containerName,
		})
		if stopErr != nil {
			c.Ui.Error(fmt.Sprintf("Failed to stop service container: %s", stopErr.Error()))
			return 1
		}

		destroyErr = container.Destroy(c.Context, container.DestroyInput{
			Name: containerName,
		})
	}

	// todo: remove any attached volumes
	removeErr := os.RemoveAll(serviceRoot)
	if removeErr != nil {
		c.Ui.Error(fmt.Sprintf("Failed to remove service data: %s", removeErr.Error()))
	}
	if destroyErr != nil {
		c.Ui.Error(fmt.Sprintf("Failed to destroy service container: %s", destroyErr.Error()))
	}
	if removeErr != nil || destroyErr != nil {
		return 1
	}

	return 0
}
