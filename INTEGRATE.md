# Integration

This doc covers how to integrate the dokku service plugins with the `dokku-service` project.

## Commands

### `service-create`

Flags:

- `-c|--config-options "--args --go=here"`:
  - description: extra arguments to pass to the container create command (default: None)
  - map: `--arguments`
- `-C|--custom-env "USER=alpha;HOST=beta"`:
  - description: semi-colon delimited environment variables to start the service with
  - map: `--env USER=alpha --env HOST=beta`
- `-i|--image IMAGE`:
  - description: the image name to start the service with
  - map: `--image-name`
- `-I|--image-version IMAGE_VERSION`:
  - description: the image version to start the service with
  - map: `--image-version`
- `-m|--memory MEMORY`:
  - description: container memory limit in megabytes (default: unlimited)
  - map: `--container-create-flags "--memory M"`
- `-N|--initial-network INITIAL_NETWORK`:
  - description: the initial network to attach the service to
  - map: `--container-create-flags "--network INITIAL_NETWORK"`
- `-p|--password PASSWORD`: override the user-level service password
  - map: TODO
- `-P|--post-create-network NETWORKS`:
  - description: a comma-separated list of networks to attach the service container to after service creation
  - map: `--post-create-network`
- `-r|--root-password PASSWORD`:
  - description: override the root-level service password
  - map: TODO
- `-S|--post-start-network NETWORKS`:
  - description: a comma-separated list of networks to attach the service container to after service start
  - map: `--post-start-network`
- `-s|--shm-size SHM_SIZE`:
  - description: override shared memory size for postgres docker container
  - map: `--container-create-flags "--shm-size M"`
