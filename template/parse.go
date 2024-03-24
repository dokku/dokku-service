package template

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/asottile/dockerfile"
	"github.com/gosimple/slug"
)

type Label string

const (
	LABEL_NAME                     Label = "com.dokku.template.name"
	LABEL_DESCRIPTION              Label = "com.dokku.template.description"
	LABEL_CONFIG_COMMANDS_CONNECT  Label = "com.dokku.template.config.commands.connect"
	LABEL_CONFIG_COMMANDS_EXPORT   Label = "com.dokku.template.config.commands.export"
	LABEL_CONFIG_COMMANDS_IMPORT   Label = "com.dokku.template.config.commands.import"
	LABEL_CONFIG_HOOKS_IMAGE       Label = "com.dokku.template.config.hooks.image"
	LABEL_CONFIG_HOOKS_PRE_CREATE  Label = "com.dokku.template.config.hooks.pre-create"
	LABEL_CONFIG_HOOKS_POST_CREATE Label = "com.dokku.template.config.hooks.post-create"
	LABEL_CONFIG_HOOKS_POST_START  Label = "com.dokku.template.config.hooks.post-start"
	LABEL_CONFIG_PORTS_EXPOSE      Label = "com.dokku.template.config.ports.expose"
	LABEL_CONFIG_PORTS_WAIT        Label = "com.dokku.template.config.ports.wait"
)

type ServiceTemplate struct {
	Name              string            `json:"name"`
	Image             ServiceImage      `json:"image"`
	Path              string            `json:"path"`
	DockerfilePath    string            `json:"dockerfile_path"`
	Description       string            `json:"description"`
	Arguments         []Argument        `json:"arguments"`
	Hooks             ServiceHooks      `json:"hooks"`
	ExportedVariables map[string]string `json:"exported_variables"`
	Commands          map[string]string `json:"commands"`
	Ports             ServicePorts      `json:"ports"`
	Volumes           []Volume          `json:"volumes"`
}

type ServiceHooks struct {
	Image      string `json:"image"`
	PreCreate  bool   `json:"pre_create"`
	PostCreate bool   `json:"post_create"`
	PostStart  bool   `json:"post_start"`
}

type ServiceImage struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type ServicePorts struct {
	Expose []int `json:"expose"`
	Wait   []int `json:"wait"`
}

type Argument struct {
	Name       string `json:"name"`
	Template   string `json:"template"`
	Value      string `json:"value"`
	IsVariable bool   `json:"is_variable"`
}

type Volume struct {
	Alias         string `json:"alias"`
	ContainerPath string `json:"container_path"`
}

var validLabels map[Label]bool

func init() {
	validLabels = map[Label]bool{
		LABEL_NAME:                     true,
		LABEL_DESCRIPTION:              true,
		LABEL_CONFIG_COMMANDS_CONNECT:  true,
		LABEL_CONFIG_COMMANDS_EXPORT:   true,
		LABEL_CONFIG_COMMANDS_IMPORT:   true,
		LABEL_CONFIG_HOOKS_IMAGE:       true,
		LABEL_CONFIG_HOOKS_PRE_CREATE:  true,
		LABEL_CONFIG_HOOKS_POST_CREATE: true,
		LABEL_CONFIG_PORTS_EXPOSE:      true,
		LABEL_CONFIG_PORTS_WAIT:        true,
	}
}

func ParseDockerfile(path string) (ServiceTemplate, error) {
	dockerfilePath := fmt.Sprintf("%s/Dockerfile", path)
	commands, err := dockerfile.ParseFile(dockerfilePath)
	if err != nil {
		return ServiceTemplate{}, fmt.Errorf("dockerfile parse error: %w", err)
	}

	arguments := []Argument{}
	volumes := []Volume{}
	for _, command := range commands {
		if err := validateLabel(command); err != nil {
			return ServiceTemplate{}, fmt.Errorf("invalid LABEL directive: %w", err)
		}

		if isArg(command) {
			argument, err := parseArg(command)
			if err != nil {
				return ServiceTemplate{}, fmt.Errorf("invalid ARG directive: %w", err)
			}
			arguments = append(arguments, argument)
		}

		if isVolume(command) {
			volume, err := parseVolume(command)
			if err != nil {
				return ServiceTemplate{}, fmt.Errorf("invalid VOLUME directive: %w", err)
			}
			volumes = append(volumes, volume)
		}
	}

	image := ServiceImage{}
	for _, argument := range arguments {
		if argument.Name == "IMAGE" {
			parts := strings.Split(argument.Value, ":")
			image.Name = parts[0]
			if len(parts) > 1 {
				image.Tag = parts[1]
			}
		}
	}

	name, err := getLabelValue(commands, LABEL_NAME)
	if err != nil {
		return ServiceTemplate{}, fmt.Errorf("missing required label %s: %w", string(LABEL_NAME), err)
	}

	description, err := getLabelValue(commands, LABEL_DESCRIPTION)
	if err != nil {
		return ServiceTemplate{}, fmt.Errorf("missing required label %s: %w", string(LABEL_DESCRIPTION), err)
	}

	hookImage := getLabelValueWithDefault(commands, LABEL_CONFIG_HOOKS_IMAGE, "bash:5")
	preCreateHook, err := strconv.ParseBool(getLabelValueWithDefault(commands, LABEL_CONFIG_HOOKS_PRE_CREATE, "false"))
	if err != nil {
		return ServiceTemplate{}, fmt.Errorf("invalid value for label label %s: %w", string(LABEL_CONFIG_HOOKS_PRE_CREATE), err)
	}

	postCreateHook, err := strconv.ParseBool(getLabelValueWithDefault(commands, LABEL_CONFIG_HOOKS_POST_CREATE, "false"))
	if err != nil {
		return ServiceTemplate{}, fmt.Errorf("invalid value for label label %s: %w", string(LABEL_CONFIG_HOOKS_POST_CREATE), err)
	}

	postStartHook, err := strconv.ParseBool(getLabelValueWithDefault(commands, LABEL_CONFIG_HOOKS_POST_START, "false"))
	if err != nil {
		return ServiceTemplate{}, fmt.Errorf("invalid value for label label %s: %w", string(LABEL_CONFIG_HOOKS_POST_START), err)
	}

	waitPorts := []int{}
	for _, port := range strings.Split(getLabelValueWithDefault(commands, LABEL_CONFIG_PORTS_WAIT, ""), ",") {
		if port == "" {
			continue
		}

		p, err := strconv.Atoi(port)
		if err != nil {
			return ServiceTemplate{}, fmt.Errorf("invalid value for label label %s: %w", string(LABEL_CONFIG_PORTS_WAIT), err)
		}

		waitPorts = append(waitPorts, p)
	}

	exposePorts := []int{}
	for _, port := range strings.Split(getLabelValueWithDefault(commands, LABEL_CONFIG_PORTS_EXPOSE, ""), ",") {
		if port == "" {
			continue
		}

		p, err := strconv.Atoi(port)
		if err != nil {
			return ServiceTemplate{}, fmt.Errorf("invalid value for label label %s: %w", string(LABEL_CONFIG_PORTS_EXPOSE), err)
		}

		exposePorts = append(exposePorts, p)
	}

	template := ServiceTemplate{
		Name:           name,
		Image:          image,
		Description:    description,
		DockerfilePath: dockerfilePath,
		Path:           path,
		Arguments:      arguments,
		Ports: ServicePorts{
			Expose: exposePorts,
			Wait:   waitPorts,
		},
		Hooks: ServiceHooks{
			Image:      hookImage,
			PreCreate:  preCreateHook,
			PostCreate: postCreateHook,
			PostStart:  postStartHook,
		},
		Volumes: volumes,
	}

	return template, nil
}

func isArg(command dockerfile.Command) bool {
	return strings.ToUpper(command.Cmd) == "ARG"
}

func isLabel(command dockerfile.Command) bool {
	return strings.ToUpper(command.Cmd) == "LABEL"
}

func isVolume(command dockerfile.Command) bool {
	return strings.ToUpper(command.Cmd) == "VOLUME"
}

func parseArg(command dockerfile.Command) (Argument, error) {
	if err := validateArgument(command); err != nil {
		return Argument{}, err
	}

	parts := strings.SplitN(command.Value[0], "=", 2)
	argument := Argument{
		Name: parts[0],
	}

	if len(parts) == 2 {
		s, err := strconv.Unquote(parts[1])
		if errors.Is(err, strconv.ErrSyntax) {
			argument.Template = parts[1]
		} else {
			argument.Template = s
		}

		tmpl, err := template.New("base").Funcs(sprig.FuncMap()).Parse(argument.Template)
		if err != nil {
			return Argument{}, fmt.Errorf("failed to parse argument template: %w", err)
		}

		builder := &strings.Builder{}
		if err := tmpl.Execute(builder, nil); err != nil {
			return Argument{}, fmt.Errorf("failed to initialize argument: %w", err)
		}

		argument.Value = builder.String()
		argument.IsVariable = argument.Template != argument.Value
	}

	return argument, nil
}

func validateArgument(command dockerfile.Command) error {
	if !isArg(command) {
		return errors.New("command directive is not ARG")
	}

	if len(command.Value) > 2 {
		return fmt.Errorf("cannot specify multiple arguments in single ARG directive")
	}

	return nil
}

func parseVolume(command dockerfile.Command) (Volume, error) {
	if err := validateVolume(command); err != nil {
		return Volume{}, err
	}

	volume := Volume{
		Alias:         strings.ToUpper(strings.ReplaceAll(slug.Make(command.Value[0]), "-", "_")),
		ContainerPath: command.Value[0],
	}

	return volume, nil
}

func validateVolume(command dockerfile.Command) error {
	if !isVolume(command) {
		return errors.New("command directive is not VOLUME")
	}

	if len(command.Value) > 1 {
		return fmt.Errorf("cannot specify multiple volumes in single VOLUME directive")
	}

	return nil
}

func validateLabel(command dockerfile.Command) error {
	if !isLabel(command) {
		return nil
	}

	if len(command.Value) == 1 {
		return fmt.Errorf("missing label value")
	}

	if len(command.Value) > 2 {
		return fmt.Errorf("cannot specify multiple labels in single label directive")
	}

	key := command.Value[0]
	// ignore non-template labels
	if !strings.HasPrefix(key, "com.dokku.template.") {
		return nil
	}

	// special-case exported and mapped variables
	isExported := strings.HasPrefix(key, "com.dokku.template.config.variables.exported.")
	isMapped := strings.HasPrefix(key, "com.dokku.template.config.variables.mapped.")
	if ok := validLabels[Label(key)]; ok {
		return nil
	}
	if !isExported && !isMapped {
		return fmt.Errorf("invalid label key: %s", key)
	}

	return nil
}

func getLabelValueWithDefault(commands []dockerfile.Command, label Label, defaultValue string) string {
	value, _ := getLabelValue(commands, label)
	if value == "" {
		value = defaultValue
	}

	return value
}
func getLabelValue(commands []dockerfile.Command, label Label) (string, error) {
	for _, command := range commands {
		if !isLabel(command) {
			continue
		}

		if command.Value[0] == string(label) {
			s, err := strconv.Unquote(command.Value[1])
			if errors.Is(err, strconv.ErrSyntax) {
				return command.Value[1], nil
			}

			return s, err
		}
	}

	return "", fmt.Errorf("label not found: %s", string(label))
}
