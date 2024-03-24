package volume

import (
	"context"
	"dokku-service/logstreamer"
	"dokku-service/template"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/alexellis/go-execute/v2"
	"github.com/gosimple/slug"
)

type CreateInput struct {
	DataRoot         string
	ServiceName      string
	Template         template.ServiceTemplate
	VolumeDescriptor template.Volume
	UseVolumes       bool
}

func Create(ctx context.Context, input CreateInput) (v Volume, err error) {
	if !input.UseVolumes {
		serviceRoot := fmt.Sprintf("%s/%s/%s", input.DataRoot, input.Template.Name, input.ServiceName)
		source := fmt.Sprintf("%s/%s", serviceRoot, input.VolumeDescriptor.Alias)
		mountType := "bind"
		if err := os.MkdirAll(filepath.Clean(source), os.ModePerm); err != nil {
			return Volume{}, errors.New("could not create volume host dir")
		}

		return Volume{
			Alias:         input.VolumeDescriptor.Alias,
			ContainerPath: input.VolumeDescriptor.ContainerPath,
			MountType:     mountType,
			Source:        source,
			MountArgs:     fmt.Sprintf("type=%s,source=%s,destination=%s", mountType, source, input.VolumeDescriptor.ContainerPath),
		}, nil
	}

	volumeName := fmt.Sprintf("dokku.%s.%s.%s", input.Template.Name, input.ServiceName, slug.Make(input.VolumeDescriptor.Alias))
	mountType := "volume"
	if ok, err := Exists(ctx, ExistsInput{Name: volumeName}); ok && err == nil {
		return Volume{
			Alias:         input.VolumeDescriptor.Alias,
			ContainerPath: input.VolumeDescriptor.ContainerPath,
			MountType:     "volume",
			Source:        volumeName,
			MountArgs:     fmt.Sprintf("type=%s,source=%s,destination=%s", mountType, volumeName, input.VolumeDescriptor.ContainerPath),
		}, nil
	}

	var mu sync.Mutex
	cmd := execute.ExecTask{
		Command: "docker",
		Args: []string{
			"volume", "create",
			"--label=org.label-schema.schema-version=1.0",
			"--label=org.label-schema.vendor=dokku",
			fmt.Sprintf("--label=com.dokku.service-name=%s", input.ServiceName),
			fmt.Sprintf("--label=com.dokku.service-type=%s", input.Template.Name),
			fmt.Sprintf("--label=com.dokku.service-container-path=%s", input.VolumeDescriptor.ContainerPath),
			fmt.Sprintf("--label=com.dokku.service-alias=%s", input.VolumeDescriptor.Alias),
			volumeName,
		},
		StreamStdio:  false,
		StdOutWriter: logstreamer.NewLogstreamer(os.Stdout, &mu),
		StdErrWriter: logstreamer.NewLogstreamer(os.Stderr, &mu),
	}

	cmd.PrintCommand = true
	res, err := cmd.Execute(ctx)
	if err != nil {
		return Volume{}, err
	}

	if res.ExitCode != 0 {
		// todo: return exit code
		return Volume{}, fmt.Errorf("non-zero exit code %d: %s", res.ExitCode, res.Stderr)
	}

	return Volume{
		Alias:         input.VolumeDescriptor.Alias,
		ContainerPath: input.VolumeDescriptor.ContainerPath,
		MountType:     mountType,
		Source:        volumeName,
		MountArgs:     fmt.Sprintf("type=%s,source=%s,destination=%s", mountType, volumeName, input.VolumeDescriptor.ContainerPath),
	}, nil
}
