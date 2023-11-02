ARG IMAGE=redis:7-alpine
FROM ${IMAGE}

VOLUME /data

LABEL com.dokku.template.name=redis
LABEL com.dokku.template.description="A template for managing redis"
LABEL com.dokku.template.config.commands.connect="redis-cli"
LABEL com.dokku.template.config.hooks.pre-create=true
LABEL com.dokku.template.config.hooks.image=bash:5.2
LABEL com.dokku.template.config.ports.expose=6379
LABEL com.dokku.template.config.ports.wait=6379
LABEL com.dokku.template.config.exported-variables.REDIS_URL="redis://:{{ .REDIS_PASSWORD }}@{{ .HOSTNAME }}:6379/{{ .REDIS_DATABASE }}"
LABEL com.dokku.template.config.exported-variables.REDIS_HOST="{{ .HOSTNAME }}"
LABEL com.dokku.template.config.exported-variables.REDIS_PASSWORD="{{ .REDIS_PASSWORD }}"
LABEL com.dokku.template.config.exported-variables.REDIS_PORT="6379"


ARG REDIS_DATABASE=0
ARG REDIS_PASSWORD="{{ randAlphaNum 32 }}"
ARG LANG=C.UTF-8 
ARG LC_ALL=C.UTF-8
