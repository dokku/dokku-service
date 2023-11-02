ARG IMAGE=postgres:16.0
FROM ${IMAGE}

VOLUME /var/lib/postgresql/data

LABEL com.dokku.template.name=postgres
LABEL com.dokku.template.description="A template for managing postgres"
LABEL com.dokku.template.config.commands.connect="psql -h localhost -U postgres {{ .POSTGRES_DATABASE }}"
LABEL com.dokku.template.config.commands.export="pg_dump -Fc --no-acl --no-owner -h localhost -U postgres -w {{ .POSTGRES_DATABASE }}"
LABEL com.dokku.template.config.commands.import="pg_restore -h localhost -cO --if-exists -d {{ .POSTGRES_DATABASE }} -U postgres -w"
LABEL com.dokku.template.config.ports.expose=5432
LABEL com.dokku.template.config.ports.wait=5432
LABEL com.dokku.template.config.exported-variables.DATABASE_URL="postgres://postgres:{{ .POSTGRES_PASSWORD }}@{{ .HOSTNAME }}:5432/{{ .POSTGRES_DATABASE }}"

ARG POSTGRES_PASSWORD="{{ randAlphaNum 32 }}"
ARG POSTGRES_DATABASE
ARG LANG=C.UTF-8 
ARG LC_ALL=C.UTF-8
