package image

import (
	"context"
	"dokku-service/argument"
	"dokku-service/logstreamer"
	"dokku-service/template"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alexellis/go-execute/v2"
)

type BuildInput struct {
	Arguments  map[string]argument.Argument
	BuildFlags []string
	Name       string
	Template   template.ServiceTemplate
}

func Build(ctx context.Context, input BuildInput) error {
	cmdArgs := []string{
		"image", "build",
		"-f", input.Template.DockerfilePath,
		"-t", input.Name,
	}

	for _, argument := range input.Arguments {
		if strings.HasSuffix(argument.Key, "_SECRET") {
			continue
		}

		cmdArgs = append(cmdArgs, "--build-arg")
		cmdArgs = append(cmdArgs, fmt.Sprintf("%s=%s", argument.Key, argument.Value))
	}

	for _, flag := range input.BuildFlags {
		cmdArgs = append(cmdArgs, "--"+flag)
	}

	cmdArgs = append(cmdArgs, fmt.Sprintf("templates/%s", input.Template.Name))

	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command:      "docker",
		Args:         cmdArgs,
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("image build for service failed: %w", err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}
