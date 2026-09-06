# syntax=docker/dockerfile:1.7

ARG RM_RELAY_ENVIRONMENT
FROM ${RM_RELAY_ENVIRONMENT} AS environment

FROM scratch
COPY --from=environment /opt/rm-relay/environment/identity.toml /identity.toml
