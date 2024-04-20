package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/container"
	"dokku-service/template"
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

	serviceRoot := fmt.Sprintf("%s/%s/%s", c.dataRoot, templateName, serviceName)
	if _, err := os.Stat(serviceRoot); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for service data existence: %s", err.Error()))
		return 1
	}

	configPath := fmt.Sprintf("%s/config.json", serviceRoot)
	if _, err := os.Stat(configPath); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for service config existence: %s", err.Error()))
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read service config: %s", err.Error()))
		return 1
	}

	parsedServiceTemplate := template.ServiceTemplate{}
	if err := json.Unmarshal(b, &parsedServiceTemplate); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to unmarshal service config: %s", err.Error()))
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
