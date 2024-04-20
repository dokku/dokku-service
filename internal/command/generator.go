//go:generate go run generator.go
package main

import (
	"bytes"
	"go/format"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/Masterminds/sprig/v3"
	"github.com/gobuffalo/flect"
)

type templateData struct {
	Name string
}

var tmpl = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(`package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type {{ .Name | camelcase }}Command struct {
	command.Meta

	// Context specifies the context to use
	Context context.Context

	// dataRoot specifies the root directory for service data
	dataRoot string

	// registryPath specifies an override path to the registry
	registryPath string
}

func (c *{{ .Name | camelcase }}Command) Name() string {
	return "{{ .Name | kebabcase }}"
}

func (c *{{ .Name | camelcase }}Command) Synopsis() string {
	return "{{ .Name | kebabcase }} command"
}

func (c *{{ .Name | camelcase }}Command) Help() string {
	return command.CommandHelp(c)
}

func (c *{{ .Name | camelcase }}Command) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"run command": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *{{ .Name | camelcase }}Command) Arguments() []command.Argument {
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

func (c *{{ .Name | camelcase }}Command) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *{{ .Name | camelcase }}Command) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *{{ .Name | camelcase }}Command) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.StringVar(&c.dataRoot, "data-root", DATA_ROOT, "the root directory for service data")
	f.StringVar(&c.registryPath, "registry-path", "", "an override path to the registry")
	return f
}

func (c *{{ .Name | camelcase }}Command) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *{{ .Name | camelcase }}Command) Run(args []string) int {
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

	templateRegistry, defferedTemplateFunc,err := templateRegistry(c.Context, c.registryPath)
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

	return 0
}
`))

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: go run generator.go SubcommandName")
	}

	commandFile := flect.Underscore(os.Args[1])
	commandFileName := filepath.Join("commands", commandFile+".go")

	f, err := os.Create(commandFileName)
	if err != nil {
		log.Fatal("Error creating command file:", err)
	}
	defer f.Close()

	templateData := templateData{
		Name: os.Args[1],
	}

	builder := &bytes.Buffer{}
	if err = tmpl.Execute(builder, templateData); err != nil {
		log.Fatal("Error executing template", err)
	}

	// Formatting generated code
	data, err := format.Source(builder.Bytes())
	if err != nil {
		log.Fatal("Error formatting generated code", err)
	}

	// Writing command file
	if err = os.WriteFile(commandFileName, data, os.ModePerm); err != nil {
		log.Fatal("Error writing blob file", err)
	}
}
