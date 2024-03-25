# Service Template

A service template is a collection of resources defining a "service". at it's base, it is comprised of a Dockerfile with specific labels. The Dockerfile is used to define the image that is used to run the service container.

## Image Building

Each `Dockerfile` _must_ start with an `ARG` directive defining an `IMAGE` argument with the default image to use, which _must_ be consumed by the `FROM` directive. This allows overriding the image used to run the service container. The following is a simple example:

```Dockerfile
ARG IMAGE=postgres:16.0
FROM ${IMAGE}
```

## Volumes

Service containers _must_ store non-ephemeral data on attached volumes. This ensures that loss of a container does not result in data-loss.

To specify a volume, add a `VOLUME` directive. The value will be used as a mount path for either attached Docker Volumes or host directories. Host directories will be created relative to the given service root.

For instance, with the following `Dockerfile`

```Dockerfile
ARG IMAGE=postgres:16.0
FROM ${IMAGE}
VOLUME /var/lib/postgresql/data
```

A `postgres` service named `test` will have the following folder on disk:

```
$DATA_ROOT/postgres/test/VAR_LIB_POSTGRESQL_DATA
```

If the service is created with a docker volume instead, then the volume will be named like so:

```
dokku.postgres.test.var_lib_postgresql_data
```

## Arguments

Arguments can be used to specify both the build-time and runtime environment, and are defined via the `ARG` directive. Their values can be templated with Golang templates, and use [sprig template functions](https://masterminds.github.io/sprig/) in addition to Go Template built-ins. When an `ARG` directive has no value specified, then the value _must_ be specified when creating the service.

By default, all arguments are supplied for the build image, with the exception of those with a suffix of `_SECRET`. These are ommitted during the build, and are made available as runtime environment variables for any generated service. When supplied to at runtime, the `_SECRET` suffix is removed.

## Labels

`LABEL` directives are used are used to define how `dokku-service` exposes functionality to a service. Labels without the `com.dokku.template.` prefix are ignored, and any labels with the `com.dokku.template.` that are not understood by `dokku-service` will result in an error.

### Required Labels

The following labels are required:

- `com.dokku.template.name`: The name of the service template. This _must_ match the folder name in the registry.
- `com.dokku.template.description`: A user-friendly description for the service template

### Command Labels

The following labels are used to define special commands to interact with the services. Label values are templated via golang templates, and variables defined in the runtime environment are available in the template call.

- `com.dokku.template.config.commands.connect`: A command to execute when connecting to a service container.
- `com.dokku.template.config.commands.export`: A command to execute that exports data from the datastore. Exported data should be output in a format that is consumable by the associated import command. Exported data should be written to `STDOUT`.
- `com.dokku.template.config.commands.import`: A command to execute that imports data into the datastore. Imported data is provided on `STDIN`.

### Port Labels

Ports are exposed via the following labels:

- `com.dokku.template.config.ports.expose`: A comma-delimited list integer values. Each value is a port that is exposed publicly from the `service-expose` command.
- `com.dokku.template.config.ports.wait`: An integer value that defines a port. When a service is created, a `TCP` check is performed against this port.

### Exported Variable Labels

When a service is "linked" to another container, a list of environment variables are exposable to those containers. Each variable has the prefix `com.dokku.template.config.variables.exported.`. Label values are templated via golang templates, and variables defined in the runtime environment are available in the template call.

With the following label:

```Dockerfile
LABEL com.dokku.template.config.variables.exported.DATABASE_URL="postgres://postgres:{{ .POSTGRES_PASSWORD_SECRET }}@{{ .HOSTNAME }}:5432/{{ .POSTGRES_DATABASE }}"
```

The variable `DATABASE_URL` would be exposed with the templated value `postgres://postgres:{{ .POSTGRES_PASSWORD_SECRET }}@{{ .HOSTNAME }}:5432/{{ .POSTGRES_DATABASE }}`.

### Mapped Variable Labels

- `com.dokku.template.config.variables.mapped.password`
- `com.dokku.template.config.variables.mapped.root-password`
