# Thought Engine. Build context: the Thought-engine source checked out at the pinned commit.
ARG GO_IMAGE=golang:1.24-alpine
ARG RUN_IMAGE=alpine:3.20
FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/thought-engine .

FROM ${RUN_IMAGE}
ARG VERSION=dev
LABEL org.opencontainers.image.title="thought-engine" org.opencontainers.image.revision="${VERSION}" \
      org.opencontainers.image.source="https://github.com/Arameair/Thought-engine"
COPY --from=build /out/thought-engine /usr/local/bin/thought-engine
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/thought-engine"]
