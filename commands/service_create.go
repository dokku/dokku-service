package commands

import (
	"context"
	"dokku-service/logstreamer"
	"dokku-service/template"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/gosimple/slug"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const DATA_ROOT = "/tmp"

type ServiceCreateCommand struct {
	command.Meta

	arguments  map[string]string
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
		c.Ui.Error("Service already exists")
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

	envFile := fmt.Sprintf("%s/%s/%s/.env", DATA_ROOT, entry.Name, serviceName)
	envLines := []string{}
	for _, argument := range containerArgs {
		envLines = append(envLines, fmt.Sprintf(`%s=%s`, argument.Key, argument.Value))
	}
	if err := os.WriteFile(envFile, []byte(strings.Join(envLines, "\n")+"\n"), 0o666); err != nil {
		c.Ui.Error("Failed to write settings for service: " + err.Error())
		return 1
	}

	cmdArgs := []string{
		"container", "create",
		"--name", containerName,
		"--env-file", envFile,
		"--restart", "always",
		"--hostname", containerName,
		"--cidfile", fmt.Sprintf("%s/%s/%s/ID", DATA_ROOT, entry.Name, serviceName),
	}
	cmdArgs = append(cmdArgs, "--label", fmt.Sprintf("com.dokku.service-volumes=%s", strconv.FormatBool(c.useVolumes)))

	logger.LogHeader2("Creating volumes")
	var createdVolumes []Volume
	for _, volumeDescriptor := range entry.Volumes {
		volume, err := c.createVolume(serviceName, entry, volumeDescriptor)
		if err != nil {
			c.Ui.Error("Failed to run volume for service: " + err.Error())
			return 1
		}

		createdVolumes = append(createdVolumes, volume)
		cmdArgs = append(cmdArgs, "--mount", volume.MountArgs)
	}
	cmdArgs = append(cmdArgs, imageName)

	logger.LogHeader2("Running pre-create hook")
	if err := c.executeHook("pre-create", entry.Hooks.PreCreate, serviceName, createdVolumes, entry); err != nil {
		c.Ui.Error("Failed to create container for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Creating container")
	if err := c.createContainer(cmdArgs); err != nil {
		c.Ui.Error("Failed to create container for service: " + err.Error())
		return 1
	}

	// todo: attach container to container-specific network
	logger.LogHeader2("Running post-create hook")
	if err := c.executeHook("post-create", entry.Hooks.PostCreate, serviceName, createdVolumes, entry); err != nil {
		c.Ui.Error("Failed to create container for service: " + err.Error())
		return 1
	}

	logger.LogHeader2("Starting container")
	if err := c.startContainer(containerName); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func (c *ServiceCreateCommand) containerExists(containerName string) (bool, error) {
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"container", "inspect",
			containerName,
		},
		StreamStdio: false,
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(context.Background())
	if err != nil {
		return false, fmt.Errorf("check for service existence failed: %w", err)
	}

	if res.ExitCode != 0 {
		return false, nil
	}

	return true, nil
}

func (c *ServiceCreateCommand) buildImage(imageName string, containerArgs map[string]Argument, template template.ServiceTemplate) error {
	cmdArgs := []string{
		"image", "build",
		"-f", template.DockerfilePath,
		"-t", imageName,
	}

	for _, argument := range containerArgs {
		cmdArgs = append(cmdArgs, "--build-arg")
		cmdArgs = append(cmdArgs, fmt.Sprintf("%s=%s", argument.Key, argument.Value))
	}
	cmdArgs = append(cmdArgs, ".")

	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:      "docker",
		Args:         cmdArgs,
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(context.Background())
	if err != nil {
		return fmt.Errorf("image build for service failed: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}

func (c *ServiceCreateCommand) createContainer(cmdArgs []string) error {
	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:      "docker",
		Args:         cmdArgs,
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(context.Background())
	if err != nil {
		return fmt.Errorf("container create for service failed: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}

func (c *ServiceCreateCommand) executeHook(hook string, hookExists bool, serviceName string, volumes []Volume, template template.ServiceTemplate) error {
	if !hookExists {
		return nil
	}

	// todo: support a library for templates
	hookPath := fmt.Sprintf("templates/%s/bin/%s", template.Name, hook)
	hookPath, err := filepath.Abs(hookPath)
	if err != nil {
		return fmt.Errorf("deriving absolute path to %s hook failed: %w", hook, err)
	}

	cmdArgs := []string{
		"container",
		"run",
		"--rm",
		"--volume",
		fmt.Sprintf("%s:/usr/local/bin/hook", hookPath),
		"--env-file",
		fmt.Sprintf("%s/%s/%s/.env", DATA_ROOT, template.Name, serviceName),
	}

	for _, volume := range volumes {
		cmdArgs = append(cmdArgs, "--mount", volume.MountArgs)
		cmdArgs = append(cmdArgs, "--env", fmt.Sprintf("VOLUME_%s=%s", volume.Alias, volume.ContainerPath))
	}

	cmdArgs = append(cmdArgs, template.Hooks.Image, "/usr/local/bin/hook")
	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:      "docker",
		Args:         cmdArgs,
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(context.Background())
	if err != nil {
		return fmt.Errorf("%s hook container for service failed: %w", hook, err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}

func (c *ServiceCreateCommand) startContainer(containerName string) error {
	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:      "docker",
		Args:         []string{"container", "start", containerName},
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(context.Background())
	if err != nil {
		return fmt.Errorf("container start for service failed: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}

type Volume struct {
	Alias         string
	Source        string
	MountType     string
	ContainerPath string
	MountArgs     string
}

func (c *ServiceCreateCommand) createVolume(serviceName string, template template.ServiceTemplate, volumeDescriptor template.Volume) (v Volume, err error) {
	if !c.useVolumes {
		source := fmt.Sprintf("%s/%s/%s/%s", DATA_ROOT, template.Name, serviceName, volumeDescriptor.Alias)
		mountType := "bind"
		if err := os.MkdirAll(filepath.Clean(source), os.ModePerm); err != nil {
			return Volume{}, errors.New("could not create volume host dir")
		}

		return Volume{
			Alias:         volumeDescriptor.Alias,
			ContainerPath: volumeDescriptor.ContainerPath,
			MountType:     mountType,
			Source:        source,
			MountArgs:     fmt.Sprintf("type=%s,source=%s,destination=%s", mountType, source, volumeDescriptor.ContainerPath),
		}, nil
	}

	var mu sync.Mutex
	source := fmt.Sprintf("dokku.%s.%s.%s", template.Name, serviceName, slug.Make(volumeDescriptor.Alias))
	mountType := "volume"
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"volume", "create",
			"--label=org.label-schema.schema-version=1.0",
			"--label=org.label-schema.vendor=dokku",
			fmt.Sprintf("--label=com.dokku.service-name=%s", serviceName),
			fmt.Sprintf("--label=com.dokku.service-type=%s", template.Name),
			fmt.Sprintf("--label=com.dokku.service-container-path=%s", volumeDescriptor.ContainerPath),
			fmt.Sprintf("--label=com.dokku.service-alias=%s", volumeDescriptor.Alias),
			source,
		},
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(context.Background())
	if err != nil {
		return Volume{}, err
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return Volume{}, fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return Volume{
		Alias:         volumeDescriptor.Alias,
		ContainerPath: volumeDescriptor.ContainerPath,
		MountType:     mountType,
		Source:        source,
		MountArgs:     fmt.Sprintf("type=%s,source=%s,destination=%s", mountType, source, volumeDescriptor.ContainerPath),
	}, nil
}

type Argument struct {
	Key      string
	Value    string
	Override bool
}

func (c *ServiceCreateCommand) collectContainerArgs(template template.ServiceTemplate) (map[string]Argument, error) {
	arguments := map[string]Argument{}

	for _, argument := range template.Arguments {
		arguments[argument.Name] = Argument{
			Key:      argument.Name,
			Value:    argument.Value,
			Override: false,
		}
	}

	for key, value := range c.arguments {
		if len(value) == 0 {
			value = os.Getenv(key)
		}

		arguments[key] = Argument{
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
