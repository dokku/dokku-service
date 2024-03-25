# Service Registry

A service registry is a collection of service templates on disk. Each service template is a folder within the registry, and the folder _must_ match the service template name.

The `dokku-service` project ships with it's own, vendored service registry. If no `registry-path` is specified, `dokku-service` will reference the internal registry, which may be updated with every `dokku-service` release.
