package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/moby/moby/client"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/argument"
	"dokku-service/container"
	"dokku-service/healthcheck"
	"dokku-service/hook"
	"dokku-service/image"
	"dokku-service/network"
	"dokku-service/service"
	"dokku-service/template"
	"dokku-service/volume"
)

const DATA_ROOT = "/tmp"

type ServiceCreateCommand struct {
	command.Meta

	// Context specifies the context to use
	Context context.Context

	// arguments specifies the arguments to pass to the service
	arguments map[string]string

	// containerCreateFlags specifies the flags to pass to the container create command
	containerCreateFlags []string

	// dataRoot specifies the root directory for service data
	dataRoot string

	// env specifies the environment variables to pass to the service
	env map[string]string

	// imageName specifies the name to use when building the image
	imageName string

	// imageTag specifies the tag to use when building the image
	imageTag string

	// imageBuildFlags specifies the flags to pass to the image build command
	imageBuildFlags []string

	// postCreateNetwork specifies the network to attach to the container after creation
	postCreateNetwork []string

	// postStartNetwork specifies the network to attach to the container after start
	postStartNetwork []string

	// registryPath specifies an override path to the registry
	registryPath string

	// trace specifies whether to output trace information
	trace bool

	// useVolumes specifies whether to use volumes or directories for service data
	useVolumes bool
}

func (c *ServiceCreateCommand) Name() string {
	return "service-create"
}

func (c *ServiceCreateCommand) Synopsis() string {
	return "Service create command"
}

func (c *ServiceCreateCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ServiceCreateCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Does nothing": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *ServiceCreateCommand) Arguments() []command.Argument {
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

func (c *ServiceCreateCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ServiceCreateCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ServiceCreateCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.StringToStringVar(&c.arguments, "argument", map[string]string{}, "arguments to set when creating the service")
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringToStringVar(&c.env, "env", map[string]string{}, "env variables to set when creating the service")
	f.StringArrayVar(&c.containerCreateFlags, "container-create-flags", []string{}, "flags to pass to the container create command")
	f.StringVar(&c.imageName, "image-name", "", "the name to use when building the image")
	f.StringVar(&c.imageTag, "image-tag", "", "the tag to use when building the image")
	f.StringArrayVar(&c.imageBuildFlags, "image-build-flags", []string{}, "flags to pass to the image build command")
	f.StringSliceVar(&c.postCreateNetwork, "post-create-network", []string{}, "network to attach to the container after creation")
	f.StringSliceVar(&c.postStartNetwork, "post-start-network", []string{}, "network to attach to the container after start")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	f.BoolVar(&c.trace, "trace", false, "output trace information")
	f.BoolVar(&c.useVolumes, "use-volumes", false, "use volumes instead of a directory on disk for data")
	return f
}

func (c *ServiceCreateCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *ServiceCreateCommand) Run(args []string) int {
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

	serviceName := arguments["name"].StringValue()

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

	// todo: ensure the service doesn't exist at the specified path

	logger.LogHeader1(fmt.Sprintf("Creating %s service %s", serviceTemplate.Name, serviceName))

	containerName := container.Name(container.NameInput{
		ServiceName: serviceName,
		ServiceType: serviceTemplate.Name,
	})
	networkAlias := network.Alias(network.AliasInput{
		ServiceName: serviceName,
		ServiceType: serviceTemplate.Name,
	})
	ok, err = c.containerExists(containerName)
	if err != nil {
		c.Ui.Error("Failed to check for existing service: " + err.Error())
		return 1
	}
	if ok {
		c.Ui.Error("Service container already exists")
		return 2
	}

	// todo: improve logging
	for _, networkName := range c.postCreateNetwork {
		ok, err = network.Exists(c.Context, network.ExistsInput{
			Name:  networkName,
			Trace: c.trace,
		})
		if err != nil {
			c.Ui.Error("Failed to check for network existence: " + err.Error())
			return 1
		}
		if !ok {
			c.Ui.Error(fmt.Sprintf("Missing post-create network: %s", networkName))
			return 1
		}
	}

	for _, networkName := range c.postStartNetwork {
		ok, err = network.Exists(c.Context, network.ExistsInput{
			Name:  networkName,
			Trace: c.trace,
		})
		if err != nil {
			c.Ui.Error("Failed to check for network existence: " + err.Error())
		}
		if !ok {
			c.Ui.Error(fmt.Sprintf("Missing post-start network: %s", networkName))
			return 1
		}
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", c.dataRoot, serviceTemplate.Name, serviceName)
	if _, err := os.Stat(serviceRoot); err != nil && !errors.Is(err, os.ErrNotExist) {
		c.Ui.Error("Service directory already exists but container is not running")
		return 3
	}

	containerArgs, err := c.collectContainerArgs(serviceTemplate)
	if err != nil {
		c.Ui.Error("Failed to collect arguments for service: " + err.Error())
		return 1
	}

	// todo: refactor to force-set the IMAGE argument or drop completely
	if c.imageName != "" {
		serviceTemplate.Image.Name = c.imageName
	}
	if c.imageTag != "" {
		serviceTemplate.Image.Tag = c.imageTag
	}
	containerArgs["IMAGE"] = argument.Argument{
		Key:      "IMAGE",
		Value:    fmt.Sprintf("%s:%s", serviceTemplate.Image.Name, serviceTemplate.Image.Tag),
		Override: true,
	}

	imageName := image.Name(image.NameInput{
		ServiceName: serviceName,
		ServiceType: serviceTemplate.Name,
	})
	logger.LogHeader2("Building base image from template")
	if err := c.buildImage(imageName, containerArgs, serviceTemplate); err != nil {
		c.Ui.Error("Failed to build image for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Writing settings for service")
	envFile := fmt.Sprintf("%s/.env", serviceRoot)
	envLines := []string{}
	envConfig := map[string]string{}
	for _, argument := range containerArgs {
		envConfig[strings.TrimSuffix(argument.Key, "_SECRET")] = argument.Value
		envLines = append(envLines, fmt.Sprintf(`%s=%s`, strings.TrimSuffix(argument.Key, "_SECRET"), argument.Value))
	}

	for key, value := range c.env {
		if _, ok := envConfig[key]; ok {
			c.Ui.Error(fmt.Sprintf("Environment variable '%s' must be set by argument", key))
			return 1
		}

		envConfig[key] = value
		envLines = append(envLines, fmt.Sprintf(`%s=%s`, key, value))
	}

	if err := os.MkdirAll(fmt.Sprintf("%s/%s", c.dataRoot, serviceTemplate.Name), os.ModePerm); err != nil {
		c.Ui.Error("Failed to create service directory: " + err.Error())
		return 1
	}

	if err := os.MkdirAll(serviceRoot, os.ModePerm); err != nil {
		c.Ui.Error("Failed to create service directory: " + err.Error())
		return 1
	}

	if err := os.WriteFile(envFile, []byte(strings.Join(envLines, "\n")+"\n"), 0o666); err != nil {
		c.Ui.Error("Failed to write settings for service: " + err.Error())
		return 1
	}

	configFile := fmt.Sprintf("%s/config.json", serviceRoot)
	createConfig := service.ConfigOutput{
		Config: service.RunConfig{
			Arguments:            c.arguments,
			ContainerCreateFlags: c.containerCreateFlags,
			DataRoot:             c.dataRoot,
			EnvironmentVariables: envConfig,
			Image: service.RunImageConfig{
				Name: c.imageName,
				Tag:  c.imageTag,
			},
			ImageBuildFlags: c.imageBuildFlags,
			ServiceRoot:     serviceRoot,
			UseVolumes:      c.useVolumes,
		},
		Template: serviceTemplate,
	}
	data, err := json.MarshalIndent(createConfig, "", "  ")
	if err != nil {
		c.Ui.Error("Failed to marshal create settings for service: " + err.Error())
		return 1
	}

	if err := os.WriteFile(configFile, data, 0o666); err != nil {
		c.Ui.Error("Failed to write create settings for service: " + err.Error())
		return 1
	}

	// todo: ensure volumes dont exist
	logger.LogHeader2("Creating volumes")
	var createdVolumes []volume.Volume
	for _, volumeDescriptor := range serviceTemplate.Volumes {
		volume, err := c.createVolume(serviceName, serviceTemplate, volumeDescriptor)
		if err != nil {
			c.Ui.Error("Failed to run volume for service: " + err.Error())
			return 1
		}

		createdVolumes = append(createdVolumes, volume)
	}

	logger.LogHeader2("Executing pre-create hook")
	if err := c.executeHook("pre-create", serviceTemplate.Hooks.PreCreate, serviceName, createdVolumes, serviceTemplate); err != nil {
		c.Ui.Error("Failed to execute pre-create hook for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Creating container")
	err = container.Create(c.Context, container.CreateInput{
		CreateFlags:   c.containerCreateFlags,
		ContainerName: containerName,
		EnvFile:       envFile,
		ImageName:     imageName,
		ServiceRoot:   serviceRoot,
		Trace:         c.trace,
		UseVolumes:    c.useVolumes,
		Volumes:       createdVolumes,
	})
	if err != nil {
		c.Ui.Error("Failed to create container for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Attaching container to post-create networks")
	for _, networkName := range c.postCreateNetwork {
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
	if err := c.executeHook("post-create", serviceTemplate.Hooks.PostCreate, serviceName, createdVolumes, serviceTemplate); err != nil {
		c.Ui.Error("Failed to execute post-create hook for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Starting container")
	if err := c.startContainer(containerName); err != nil {
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
		Ports:        serviceTemplate.Ports.Wait,
		Timeout:      5,
		Trace:        c.trace,
		Wait:         1,
	}); err != nil {
		c.Ui.Error("Failed to wait for service to be ready: " + err.Error())
		return 1
	}

	logger.LogHeader2("Attaching container to post-start networks")
	for _, networkName := range c.postStartNetwork {
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
	if err := c.executeHook("post-start", serviceTemplate.Hooks.PostStart, serviceName, createdVolumes, serviceTemplate); err != nil {
		c.Ui.Error("Failed to execute post-start hook for service: " + err.Error())
		return 1
	}

	return 0
}

func (c *ServiceCreateCommand) containerExists(containerName string) (bool, error) {
	return container.Exists(c.Context, container.ExistsInput{
		Name:  containerName,
		Trace: c.trace,
	})
}

func (c *ServiceCreateCommand) buildImage(imageName string, containerArgs map[string]argument.Argument, template template.ServiceTemplate) error {
	return image.Build(c.Context, image.BuildInput{
		Arguments:  containerArgs,
		BuildFlags: c.imageBuildFlags,
		Name:       imageName,
		Template:   template,
		Trace:      c.trace,
	})
}

func (c *ServiceCreateCommand) executeHook(name string, hookExists bool, serviceName string, volumes []volume.Volume, template template.ServiceTemplate) error {
	return hook.Execute(c.Context, hook.ExecuteInput{
		DataRoot:    c.dataRoot,
		Exists:      hookExists,
		Name:        name,
		ServiceName: serviceName,
		Template:    template,
		Volumes:     volumes,
		Trace:       c.trace,
	})
}

func (c *ServiceCreateCommand) startContainer(containerName string) error {
	return container.Start(c.Context, container.StartInput{
		Name:  containerName,
		Trace: c.trace,
	})
}

func (c *ServiceCreateCommand) createVolume(serviceName string, template template.ServiceTemplate, volumeDescriptor template.Volume) (v volume.Volume, err error) {
	return volume.Create(c.Context, volume.CreateInput{
		DataRoot:         c.dataRoot,
		ServiceName:      serviceName,
		Template:         template,
		Trace:            c.trace,
		UseVolumes:       c.useVolumes,
		VolumeDescriptor: volumeDescriptor,
	})
}

func (c *ServiceCreateCommand) collectContainerArgs(template template.ServiceTemplate) (map[string]argument.Argument, error) {
	arguments := map[string]argument.Argument{}

	for _, templateArg := range template.Arguments {
		arguments[templateArg.Name] = argument.Argument{
			Key:      templateArg.Name,
			Value:    templateArg.Value,
			Override: false,
		}
	}

	for key, value := range c.arguments {
		if len(value) == 0 {
			value = os.Getenv(key)
		}

		arguments[key] = argument.Argument{
			Key:      key,
			Value:    value,
			Override: true,
		}
	}

	for _, argument := range arguments {
		if argument.Value == "" && !argument.Override {
			return arguments, fmt.Errorf("missing value for required service argument %s", argument.Key)
		}
	}

	return arguments, nil
}
