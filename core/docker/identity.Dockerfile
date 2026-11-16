# tass-identity (Identity repo, Go). Build context: the identity source checked out at the pinned commit.
# No entity, credential, passphrase or service key is built in: each installation creates its own at run time.
ARG GO_IMAGE=golang:1.24-alpine
ARG RUN_IMAGE=alpine:3.20
FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/tass-identity ./cmd/tass-identity

FROM ${RUN_IMAGE}
ARG VERSION=dev
LABEL org.opencontainers.image.title="tass-identity" org.opencontainers.image.revision="${VERSION}" \
      org.opencontainers.image.source="https://github.com/Arameair/identity"
COPY --from=build /out/tass-identity /usr/local/bin/tass-identity
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/tass-identity"]
