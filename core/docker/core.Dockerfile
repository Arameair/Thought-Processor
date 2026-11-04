# tass-core (Thought-Processor's Go core). Build context: core/
# Multi-stage, static binary, non-root, no secrets: tokens and keys are mounted at run time, never built in.
ARG GO_IMAGE=golang:1.24-alpine
ARG RUN_IMAGE=alpine:3.20
FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY go.mod ./
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/tass-core ./cmd/tass-core

FROM ${RUN_IMAGE}
ARG VERSION=dev
LABEL org.opencontainers.image.title="tass-core" org.opencontainers.image.revision="${VERSION}" \
      org.opencontainers.image.source="https://github.com/Arameair/Thought-Processor"
COPY --from=build /out/tass-core /usr/local/bin/tass-core
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/tass-core"]
