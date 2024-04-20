# dokku-service

A prototype for managing dokku services based on template repositories.

## Impetus

Managing each of the service plugins is a lot of work, both in duplicating code and implementing new functionality.
This project's goal is to centralize all of that by using the Dockerfile format as the source of truth for a service.
While this won't be 100% compatible with existing services, it can be used as a base for built-in service functionality.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0
```

## Usage

```
Usage: dokku-service [--version] [--help] <command> [<args>]

Available commands are:
    service-create    Service create command
    template-info     template-info command
    template-list     template-list command
    version           Return the version of the binary
```

Status:

- [ ] service-backup
- [x] service-create
- [ ] service-clone
- [ ] service-connect
- [x] service-destroy
- [ ] service-enter
- [ ] service-exists
- [ ] service-export
- [ ] service-expose
- [ ] service-import
- [ ] service-info
- [ ] service-link
- [ ] service-linked
- [ ] service-links
- [x] service-list
- [x] service-logs
- [ ] service-pause
- [ ] service-promote
- [ ] service-restart
- [ ] service-set
- [ ] service-start
- [x] service-stop
- [ ] service-unexpose
- [ ] service-unlink
- [ ] service-upgrade

- [ ] app-links
