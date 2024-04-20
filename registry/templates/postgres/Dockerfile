ARG IMAGE=postgres:16.0
FROM ${IMAGE}

VOLUME /var/lib/postgresql/data

LABEL com.dokku.template.name=postgres
LABEL com.dokku.template.description="A template for managing postgres"
LABEL com.dokku.template.config.commands.connect="psql -h localhost -U postgres {{ .POSTGRES_DB }}"
LABEL com.dokku.template.config.commands.export="pg_dump -Fc --no-acl --no-owner -h localhost -U postgres -w {{ .POSTGRES_DB }}"
LABEL com.dokku.template.config.commands.import="pg_restore -h localhost -cO --if-exists -d {{ .POSTGRES_DB }} -U postgres -w"
LABEL com.dokku.template.config.ports.expose=5432
LABEL com.dokku.template.config.ports.wait=5432
LABEL com.dokku.template.config.variables.exported.DATABASE_URL="postgres://postgres:{{ .POSTGRES_PASSWORD_SECRET }}@{{ .HOSTNAME }}:5432/{{ .POSTGRES_DB }}"
LABEL com.dokku.template.config.variables.mapped.name="POSTGRES_DB"
LABEL com.dokku.template.config.variables.mapped.password="POSTGRES_PASSWORD"
LABEL com.dokku.template.config.variables.mapped.root-password=""



ARG POSTGRES_DB
ARG POSTGRES_PASSWORD_SECRET="{{ randAlphaNum 32 }}"
ARG LANG=C.UTF-8 
ARG LC_ALL=C.UTF-8
