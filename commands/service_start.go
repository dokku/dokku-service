package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/moby/moby/client"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/container"
	"dokku-service/healthcheck"
	"dokku-service/hook"
	"dokku-service/image"
	"dokku-service/network"
	"dokku-service/service"
	"dokku-service/volume"
)

type ServiceStartCommand struct {
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

func (c *ServiceStartCommand) Name() string {
	return "service-start"
}

func (c *ServiceStartCommand) Synopsis() string {
	return "service-start command"
}

func (c *ServiceStartCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServiceStartCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"run command": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *ServiceStartCommand) Arguments() []command.Argument {
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

func (c *ServiceStartCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServiceStartCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServiceStartCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.trace, "trace", false, "output trace information")
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	return f
}

func (c *ServiceStartCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServiceStartCommand) Run(args []string) int {
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
	networkAlias := network.Alias(network.AliasInput{
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

	if containerExists {
		cli, err := client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		dockerContainer, err := cli.ContainerInspect(c.Context, containerName)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		if dockerContainer.State.Running {
			c.Ui.Info(fmt.Sprintf("Service %s is already running", serviceName))
			return 0
		}

		c.Ui.Info(fmt.Sprintf("Service %s is not running but container exists, starting container", serviceName))
		err = container.Start(c.Context, container.StartInput{
			Name:  containerName,
			Trace: c.trace,
		})
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start container: %s", err.Error()))
			return 1
		}

		return 0
	}

	config, err := service.Config(c.Context, service.ConfigInput{
		DataRoot:    c.dataRoot,
		Name:        serviceName,
		Registry:    templateRegistry,
		ServiceType: serviceTemplate.Name,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to fetch service config: %s", err.Error()))
		return 1
	}

	err = os.RemoveAll(filepath.Join(config.Config.ServiceRoot, "ID"))
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to remove ID file: %s", err.Error()))
		return 1
	}

	// check if image exists
	imageName := image.Name(image.NameInput{
		ServiceName: serviceName,
		ServiceType: serviceTemplate.Name,
	})
	imageExists, err := image.Exists(c.Context, image.ExistsInput{
		Name:  imageName,
		Trace: c.trace,
	})
	if err != nil {
		c.Ui.Error("Failed to check for image existence: " + err.Error())
		return 1
	}

	if !imageExists {
		logger.LogHeader2("Building base image from template")
		err = image.Build(c.Context, image.BuildInput{
			Arguments:  config.Config.Arguments,
			BuildFlags: config.Config.ImageBuildFlags,
			Name:       imageName,
			Template:   config.Template,
			Trace:      c.trace,
		})
		if err != nil {
			c.Ui.Error("Failed to build image for service: " + err.Error())
			return 1
		}
	}

	logger.LogHeader2("Creating volumes")
	var createdVolumes []volume.Volume
	for _, volumeDescriptor := range config.Template.Volumes {
		volume, err := volume.Create(c.Context, volume.CreateInput{
			DataRoot:         config.Config.DataRoot,
			ServiceName:      serviceName,
			Template:         config.Template,
			Trace:            c.trace,
			UseVolumes:       config.Config.UseVolumes,
			VolumeDescriptor: volumeDescriptor,
		})
		if err != nil {
			c.Ui.Error("Failed to run volume for service: " + err.Error())
			return 1
		}

		createdVolumes = append(createdVolumes, volume)
	}

	logger.LogHeader2("Executing pre-create hook")
	err = hook.Execute(c.Context, hook.ExecuteInput{
		DataRoot:    config.Config.DataRoot,
		Exists:      config.Template.Hooks.PreCreate,
		Name:        "pre-create",
		ServiceName: serviceName,
		Template:    serviceTemplate, // use the registry's template to avoid using the temp path from config.Template
		Volumes:     createdVolumes,
		Trace:       c.trace,
	})
	if err != nil {
		c.Ui.Error("Failed to execute pre-create hook for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Creating container")
	envFile := fmt.Sprintf("%s/.env", config.Config.ServiceRoot)
	err = container.Create(c.Context, container.CreateInput{
		CreateFlags:   config.Config.ContainerCreateFlags,
		ContainerName: containerName,
		EnvFile:       envFile,
		ImageName:     imageName,
		ServiceRoot:   config.Config.ServiceRoot,
		Trace:         c.trace,
		UseVolumes:    config.Config.UseVolumes,
		Volumes:       createdVolumes,
	})
	if err != nil {
		c.Ui.Error("Failed to create container for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Attaching container to post-create networks")
	for _, networkName := range config.Config.PostCreateNetworks {
		if err := network.Connect(c.Context, network.ConnectInput{
			ContainerName: containerName,
			NetworkAlias:  networkAlias,
			NetworkName:   networkName,
			Trace:         c.trace,
		}); err != nil {
			c.Ui.Error("Failed to attach container to network: " + err.Error())
			return 1
		}
	}

	// todo: attach container to container-specific network
	logger.LogHeader2("Executing post-create hook")
	err = hook.Execute(c.Context, hook.ExecuteInput{
		DataRoot:    config.Config.DataRoot,
		Exists:      config.Template.Hooks.PostCreate,
		Name:        "post-create",
		ServiceName: serviceName,
		Template:    serviceTemplate, // use the registry's template to avoid using the temp path from config.Template
		Volumes:     createdVolumes,
		Trace:       c.trace,
	})
	if err != nil {
		c.Ui.Error("Failed to execute post-create hook for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Starting container")
	err = container.Start(c.Context, container.StartInput{
		Name:  containerName,
		Trace: c.trace,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	logger.LogHeader2("Waiting for service to be ready")
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}

	container, err := cli.ContainerInspect(c.Context, containerName)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}
	if err := healthcheck.ListeningCheck(c.Context, healthcheck.ListeningCheckInput{
		Container:    container,
		NetworkAlias: networkAlias,
		Ports:        config.Template.Ports.Wait,
		Timeout:      5,
		Trace:        c.trace,
		Wait:         1,
	}); err != nil {
		c.Ui.Error("Failed to wait for service to be ready: " + err.Error())
		return 1
	}

	logger.LogHeader2("Attaching container to post-start networks")
	for _, networkName := range config.Config.PostStartNetworks {
		if err := network.Connect(c.Context, network.ConnectInput{
			ContainerName: containerName,
			NetworkAlias:  networkAlias,
			NetworkName:   networkName,
			Trace:         c.trace,
		}); err != nil {
			c.Ui.Error("Failed to attach container to network: " + err.Error())
			return 1
		}
	}

	logger.LogHeader2("Executing post-start hook")
	err = hook.Execute(c.Context, hook.ExecuteInput{
		DataRoot:    config.Config.DataRoot,
		Exists:      config.Template.Hooks.PostStart,
		Name:        "post-start",
		ServiceName: serviceName,
		Template:    serviceTemplate, // use the registry's template to avoid using the temp path from config.Template
		Volumes:     createdVolumes,
		Trace:       c.trace,
	})
	if err != nil {
		c.Ui.Error("Failed to execute post-start hook for service: " + err.Error())
		return 1
	}

	return 0
}
