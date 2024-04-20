package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/container"
	"dokku-service/service"
)

type ServiceEnterCommand struct {
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

func (c *ServiceEnterCommand) Name() string {
	return "service-enter"
}

func (c *ServiceEnterCommand) Synopsis() string {
	return "Service enter command"
}

func (c *ServiceEnterCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServiceEnterCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Does nothing": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *ServiceEnterCommand) Arguments() []command.Argument {
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

func (c *ServiceEnterCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServiceEnterCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServiceEnterCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	f.BoolVar(&c.trace, "trace", false, "output trace information")
	return f
}

func (c *ServiceEnterCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServiceEnterCommand) Run(args []string) int {
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
	logger.LogHeader1(fmt.Sprintf("Entering %s service %s", templateName, serviceName))

	containerName := container.Name(container.NameInput{
		ServiceName: serviceName,
		ServiceType: templateName,
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
		c.Ui.Error(fmt.Sprintf("Service container %s does not exist", containerName))
		return 1
	}

	_, err = service.Config(c.Context, service.ConfigInput{
		DataRoot:    c.dataRoot,
		Name:        serviceName,
		Registry:    templateRegistry,
		ServiceType: templateName,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to fetch service config: %s", err.Error()))
		return 1
	}

	shell := "/bin/bash"
	if serviceTemplate.Commands["enter"] != "" {
		shell = serviceTemplate.Commands["enter"]
	}
	err = container.Enter(c.Context, container.EnterInput{
		Name:  containerName,
		Shell: shell,
		Trace: c.trace,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to enter container: %s", err.Error()))
		return 1
	}

	return 0
}
