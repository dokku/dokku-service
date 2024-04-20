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

type ServiceLogsCommand struct {
	command.Meta

	// Context specifies the context to use
	Context context.Context

	// dataRoot specifies the root directory for service data
	dataRoot string

	// follow specifies whether to follow the logs
	follow bool

	// registryPath specifies an override path to the registry
	registryPath string

	// tail specifies the number of lines to show from the end of the logs
	tail int
}

func (c *ServiceLogsCommand) Name() string {
	return "service-logs"
}

func (c *ServiceLogsCommand) Synopsis() string {
	return "service-logs command"
}

func (c *ServiceLogsCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServiceLogsCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"print the most recent log(s) for this service": fmt.Sprintf("%s %s postgres test-postgres", appName, c.Name()),
	}
}

func (c *ServiceLogsCommand) Arguments() []command.Argument {
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

func (c *ServiceLogsCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServiceLogsCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServiceLogsCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.follow, "follow", false, "do not stop when end of the logs are reached and wait for additional output")
	f.IntVar(&c.tail, "tail", -1, "number of lines to show from the end of the logs")
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	return f
}

func (c *ServiceLogsCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServiceLogsCommand) Run(args []string) int {
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

	_, ok := c.Ui.(*command.ZerologUi)
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
		Name: containerName,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for container existence: %s", err.Error()))
		return 1
	}

	if !containerExists {
		c.Ui.Error(fmt.Sprintf("%s service %s does not exist", serviceTemplate.Name, serviceName))
		return 1
	}

	err = container.Logs(c.Context, container.LogsInput{
		Follow: c.follow,
		Name:   containerName,
		Tail:   c.tail,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}
