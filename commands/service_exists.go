package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/service"
)

type ServiceExistsCommand struct {
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

func (c *ServiceExistsCommand) Name() string {
	return "service-exists"
}

func (c *ServiceExistsCommand) Synopsis() string {
	return "service-exists command"
}

func (c *ServiceExistsCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServiceExistsCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"run command": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *ServiceExistsCommand) Arguments() []command.Argument {
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

func (c *ServiceExistsCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServiceExistsCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServiceExistsCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.trace, "trace", false, "output trace information")
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	return f
}

func (c *ServiceExistsCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServiceExistsCommand) Run(args []string) int {
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
	exists := service.Exists(c.Context, service.ExistsInput{
		DataRoot:    c.dataRoot,
		Name:        serviceName,
		Registry:    templateRegistry,
		ServiceType: serviceTemplate.Name,
	})
	if !exists {
		c.Ui.Error(fmt.Sprintf("%s service %s does not exist", templateName, serviceName))
		return 1
	}

	logger.LogHeader1(fmt.Sprintf("%s service %s exists", templateName, serviceName))

	return 0
}
