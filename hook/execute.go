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

// ExecuteInput contains the input parameters for the Execute function
type ExecuteInput struct {
	// DataRoot specifies the root directory for the service data
	DataRoot string

	// Exists specifies if the hook exists
	Exists bool

	// Name of the hook to execute
	Name string

	// ServiceName specifies the name of the service
	ServiceName string

	// Template to use for executing the hook
	Template template.ServiceTemplate

	// TemplatePath specifies the path to the template
	TemplatePath string

	// Volumes specifies the volumes to use when executing the hook
	Volumes []volume.Volume
}

// Execute executes a hook
func Execute(ctx context.Context, input ExecuteInput) error {
	if !input.Exists {
		return nil
	}

	// todo: support a library for templates
	hookPath := filepath.Join(input.TemplatePath, "bin", input.Name)
	hookPath, err := filepath.Abs(hookPath)
	if err != nil {
		return fmt.Errorf("deriving absolute path to %s hook failed: %w", input.Name, err)
	}

	// todo: validate that hook is executable
	if err := os.Chmod(hookPath, 0755); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
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
