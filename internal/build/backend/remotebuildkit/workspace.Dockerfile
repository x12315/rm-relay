# syntax=docker/dockerfile:1.7
ARG RM_RELAY_ENVIRONMENT
FROM ${RM_RELAY_ENVIRONMENT} AS workspace-build
WORKDIR /workspace
COPY . .
RUN rm -rf /workspace/build /workspace/install
ARG RM_RELAY_MISE_CONFIGS
ARG RM_RELAY_MISE_TASK
ARG RM_RELAY_BUILD_PRESET
ARG RM_RELAY_CCACHE_ID
ENV MISE_OVERRIDE_CONFIG_FILENAMES=${RM_RELAY_MISE_CONFIGS}
ENV MISE_TASK_RUN_AUTO_INSTALL=false
ENV RM_RELAY_BUILD_PRESET=${RM_RELAY_BUILD_PRESET}
ENV RM_RELAY_OUTPUT_DIR=/rm-relay-output
RUN --mount=type=cache,id=${RM_RELAY_CCACHE_ID},target=/cache/ccache \
    CCACHE_DIR=/cache/ccache mise --locked run ${RM_RELAY_MISE_TASK}

FROM scratch AS build-output
COPY --from=workspace-build /rm-relay-output/ /
