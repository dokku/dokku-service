package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/ambassador"
	"dokku-service/container"
)

type ServicePauseCommand struct {
	command.Meta

	// Context specifies the context to use
	Context context.Context

	// dataRoot specifies the root directory for service data
	dataRoot string

	// registryPath specifies an override path to the registry
	registryPath string

	// trace specifies whether to output trace information
	trace bool
}

func (c *ServicePauseCommand) Name() string {
	return "service-pause"
}

func (c *ServicePauseCommand) Synopsis() string {
	return "service-pause command"
}

func (c *ServicePauseCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServicePauseCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"run command": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *ServicePauseCommand) Arguments() []command.Argument {
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

func (c *ServicePauseCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServicePauseCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServicePauseCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.trace, "trace", false, "output trace information")
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	return f
}

func (c *ServicePauseCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServicePauseCommand) Run(args []string) int {
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

	templateRegistry, defferedTemplateFunc, err := fetchTemplateRegistry(c.Context, c.registryPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	defer defferedTemplateFunc()

	templateName := arguments["template"].StringValue()
	serviceTemplate, err := fetchTemplate(templateRegistry, templateName)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	serviceName := arguments["name"].StringValue()
	containerName := container.Name(container.NameInput{
		ServiceName: serviceName,
		ServiceType: serviceTemplate.Name,
	})
	containerExists, err := container.Exists(c.Context, container.ExistsInput{
		Name:  containerName,
		Trace: c.trace,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for container existence: %s", err.Error()))
		return 1
	}

	if !containerExists {
		c.Ui.Error(fmt.Sprintf("%s service %s is already paused", serviceTemplate.Name, serviceName))
		return 1
	}

	logger.LogHeader1("Pausing service")
	exists, err := ambassador.Exists(c.Context, ambassador.ExistsInput{
		Name:  containerName,
		Trace: c.trace,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for ambassador existence: %s", err.Error()))
		return 1
	}
	if exists {
		err = ambassador.Stop(c.Context, ambassador.StopInput{
			Name:  containerName,
			Trace: c.trace,
		})
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to pause ambassador: %s", err.Error()))
			return 1
		}
		logger.Info("Ambassador container paused")
	}

	logger.LogHeader1("Pausing service container")
	err = container.Stop(c.Context, container.StopInput{
		Name:  containerName,
		Trace: c.trace,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to pause service: %s", err.Error()))
		return 1
	}

	logger.Info("Service container paused")
	return 0
}
