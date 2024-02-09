package commands

import (
	"context"
	"dokku-service/container"
	"dokku-service/template"
	"dokku-service/volume"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"dokku-service/argument"
	"dokku-service/hook"
	"dokku-service/image"
)

const DATA_ROOT = "/tmp"

type CreateConfig struct {
	Config   RunConfig                `json:"config"`
	Template template.ServiceTemplate `json:"template"`
}

type RunConfig struct {
	Arguments            map[string]string `json:"arguments"`
	ContainerCreateFlags []string          `json:"container_create_flags"`
	DataRoot             string            `json:"data_root"`
	EnvironmentVariables map[string]string `json:"env"`
	ImageBuildFlags      []string          `json:"image_build_flags"`
	ServiceRoot          string            `json:"service_root"`
	UseVolumes           bool              `json:"use_volumes"`
}

type ServiceCreateCommand struct {
	command.Meta

	// Context passed to the command
	Context context.Context

	// Arguments to pass to the docker container via BUILD_ARGS
	arguments map[string]string

	// Flags to pass to the container create command
	containerCreateFlags []string

	// The root directory for service data
	dataRoot string

	// Flags to pass to the image build command
	imageBuildFlags []string

	// Use volumes instead of a directory on disk for data
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
	f.StringArrayVar(&c.containerCreateFlags, "container-create-flags", []string{}, "flags to pass to the container create command")
	f.StringArrayVar(&c.imageBuildFlags, "image-build-flags", []string{}, "flags to pass to the image build command")
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

	templateName := arguments["template"].StringValue()
	serviceName := arguments["name"].StringValue()

	logger.LogHeader1(fmt.Sprintf("Creating %s service %s", templateName, serviceName))
	entry, err := template.ParseDockerfile(fmt.Sprintf("templates/%s", templateName))
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Template parse failure: %s", err.Error()))
		return 1
	}

	containerName := fmt.Sprintf("dokku.%s.%s", entry.Name, serviceName)
	ok, err = c.containerExists(containerName)
	if err != nil {
		c.Ui.Error("Failed to check for existing service: " + err.Error())
		return 1
	}
	if ok {
		c.Ui.Error("Service container already exists")
		return 2
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", c.dataRoot, entry.Name, serviceName)
	if _, err := os.Stat(serviceRoot); err != nil && !errors.Is(err, os.ErrNotExist) {
		c.Ui.Error("Service directory already exists but container is not running")
		return 3
	}

	if err := os.MkdirAll(fmt.Sprintf("%s/%s", c.dataRoot, entry.Name), os.ModePerm); err != nil {
		c.Ui.Error("Failed to create service directory: " + err.Error())
		return 1
	}

	if err := os.MkdirAll(serviceRoot, os.ModePerm); err != nil {
		c.Ui.Error("Failed to create service directory: " + err.Error())
		return 1
	}

	containerArgs, err := c.collectContainerArgs(entry)
	if err != nil {
		c.Ui.Error("Failed to collect arguments for service: " + err.Error())
		return 1
	}

	imageName := fmt.Sprintf("dokku/service-%s:latest", entry.Name)
	logger.LogHeader2("Building base image from template")
	if err := c.buildImage(imageName, containerArgs, entry); err != nil {
		c.Ui.Error("Failed to build image for service: " + err.Error())
		return 1
	}

	envFile := fmt.Sprintf("%s/.env", serviceRoot)
	envLines := []string{}
	envConfig := map[string]string{}
	for _, argument := range containerArgs {
		envConfig[strings.TrimSuffix(argument.Key, "_SECRET")] = argument.Value
		envLines = append(envLines, fmt.Sprintf(`%s=%s`, strings.TrimSuffix(argument.Key, "_SECRET"), argument.Value))
	}
	if err := os.WriteFile(envFile, []byte(strings.Join(envLines, "\n")+"\n"), 0o666); err != nil {
		c.Ui.Error("Failed to write settings for service: " + err.Error())
		return 1
	}

	configFile := fmt.Sprintf("%s/config.json", serviceRoot)
	createConfig := CreateConfig{
		Config: RunConfig{
			Arguments:            c.arguments,
			ContainerCreateFlags: c.containerCreateFlags,
			DataRoot:             c.dataRoot,
			EnvironmentVariables: envConfig,
			ImageBuildFlags:      c.imageBuildFlags,
			ServiceRoot:          serviceRoot,
			UseVolumes:           c.useVolumes,
		},
		Template: entry,
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
	for _, volumeDescriptor := range entry.Volumes {
		volume, err := c.createVolume(serviceName, entry, volumeDescriptor)
		if err != nil {
			c.Ui.Error("Failed to run volume for service: " + err.Error())
			return 1
		}

		createdVolumes = append(createdVolumes, volume)
	}

	logger.LogHeader2("Executing pre-create hook")
	if err := c.executeHook("pre-create", entry.Hooks.PreCreate, serviceName, createdVolumes, entry); err != nil {
		c.Ui.Error("Failed to execute pre-create hook for service: " + err.Error())
		return 1
	}

	err = container.Create(c.Context, container.CreateInput{
		CreateFlags:   c.containerCreateFlags,
		ContainerName: containerName,
		EnvFile:       envFile,
		ImageName:     imageName,
		ServiceRoot:   serviceRoot,
		Volumes:       createdVolumes,
		UseVolumes:    c.useVolumes,
	})

	logger.LogHeader2("Creating container")
	if err != nil {
		c.Ui.Error("Failed to create container for service: " + err.Error())
		return 1
	}

	// todo: attach container to container-specific network
	logger.LogHeader2("Executing post-create hook")
	if err := c.executeHook("post-create", entry.Hooks.PostCreate, serviceName, createdVolumes, entry); err != nil {
		c.Ui.Error("Failed to execute post-create hook for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Starting container")
	if err := c.startContainer(containerName); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// todo: wait until container is ready

	logger.LogHeader2("Executing post-start hook")
	if err := c.executeHook("post-start", entry.Hooks.PostStart, serviceName, createdVolumes, entry); err != nil {
		c.Ui.Error("Failed to execute post-start hook for service: " + err.Error())
		return 1
	}

	return 0
}

func (c *ServiceCreateCommand) containerExists(containerName string) (bool, error) {
	return container.Exists(c.Context, container.ExistsInput{
		Name: containerName,
	})
}

func (c *ServiceCreateCommand) buildImage(imageName string, containerArgs map[string]argument.Argument, template template.ServiceTemplate) error {
	return image.Build(c.Context, image.BuildInput{
		Arguments:  containerArgs,
		BuildFlags: c.imageBuildFlags,
		Name:       imageName,
		Template:   template,
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
	})
}

func (c *ServiceCreateCommand) startContainer(containerName string) error {
	return container.Start(c.Context, container.StartInput{
		Name: containerName,
	})
}

func (c *ServiceCreateCommand) createVolume(serviceName string, template template.ServiceTemplate, volumeDescriptor template.Volume) (v volume.Volume, err error) {
	return volume.Create(c.Context, volume.CreateInput{
		DataRoot:         c.dataRoot,
		ServiceName:      serviceName,
		Template:         template,
		VolumeDescriptor: volumeDescriptor,
		UseVolumes:       c.useVolumes,
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
