package hook

import (
	"context"
	"dokku-service/logstreamer"
	"dokku-service/template"
	"dokku-service/volume"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/alexellis/go-execute/v2"
)

type ExecuteInput struct {
	DataRoot    string
	Exists      bool
	Name        string
	ServiceName string
	Template    template.ServiceTemplate
	Volumes     []volume.Volume
}

func Execute(ctx context.Context, input ExecuteInput) error {
	if !input.Exists {
		return nil
	}

	// todo: support a library for templates
	hookPath := fmt.Sprintf("templates/%s/bin/%s", input.Template.Name, input.Name)
	hookPath, err := filepath.Abs(hookPath)
	if err != nil {
		return fmt.Errorf("deriving absolute path to %s hook failed: %w", input.Name, err)
	}

	serviceRoot := fmt.Sprintf("%s/%s/%s", input.DataRoot, input.Template.Name, input.ServiceName)
	cmdArgs := []string{
		"container",
		"run",
		"--rm",
		"--volume",
		fmt.Sprintf("%s:/usr/local/bin/hook", hookPath),
		"--env-file",
		fmt.Sprintf("%s/.env", serviceRoot),
	}

	for _, volume := range input.Volumes {
		cmdArgs = append(cmdArgs, "--mount", volume.MountArgs)
		cmdArgs = append(cmdArgs, "--env", fmt.Sprintf("VOLUME_%s=%s", volume.Alias, volume.ContainerPath))
	}

	cmdArgs = append(cmdArgs, input.Template.Hooks.Image, "/usr/local/bin/hook")
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
		return fmt.Errorf("%s hook container for service failed: %w", input.Name, err)
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return nil
}
