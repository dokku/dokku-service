ARG IMAGE=redis:7-alpine
FROM ${IMAGE}

VOLUME /data

LABEL com.dokku.template.name=redis
LABEL com.dokku.template.description="A template for managing redis"
LABEL com.dokku.template.config.commands.connect="redis-cli -a {{ $REDIS_PASSWORD_SECRET }}"
LABEL com.dokku.template.config.commands.enter="/bin/sh"
LABEL com.dokku.template.config.hooks.pre-create=true
LABEL com.dokku.template.config.hooks.image=bash:5.2
LABEL com.dokku.template.config.ports.expose=6379
LABEL com.dokku.template.config.ports.wait=6379
LABEL com.dokku.template.config.variables.exported.REDIS_URL="redis://:{{ .REDIS_PASSWORD_SECRET }}@{{ .HOSTNAME }}:6379/{{ .REDIS_DATABASE }}"
LABEL com.dokku.template.config.variables.exported.REDIS_HOST="{{ .HOSTNAME }}"
LABEL com.dokku.template.config.variables.exported.REDIS_PASSWORD="{{ .REDIS_PASSWORD_SECRET }}"
LABEL com.dokku.template.config.variables.exported.REDIS_PORT="6379"


ARG REDIS_DATABASE=0
ARG REDIS_PASSWORD_SECRET="{{ randAlphaNum 32 }}"
ARG LANG=C.UTF-8 
ARG LC_ALL=C.UTF-8
