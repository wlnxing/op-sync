# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.22-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod ./
RUN go mod download

COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
ARG TARGETVARIANT
RUN set -eux; \
	goos="${TARGETOS:-linux}"; \
	goarch="${TARGETARCH:-$(go env GOARCH)}"; \
	if [ "$goarch" = "arm" ] && [ -n "$TARGETVARIANT" ]; then \
		export GOARM="${TARGETVARIANT#v}"; \
	fi; \
	CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o /out/openlist-sync ./cmd/openlist-sync

FROM scratch

WORKDIR /config

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/openlist-sync /usr/local/bin/openlist-sync

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/openlist-sync"]
