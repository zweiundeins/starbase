# syntax=docker/dockerfile:1
# Pure Go (SQLite via modernc.org/sqlite): cross-compile on the build
# platform, no cgo, no emulation needed for multi-arch images.
FROM --platform=$BUILDPLATFORM golang:1.27.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/starbase ./cmd/starbase \
	&& mkdir -p /out/data

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/starbase /starbase
COPY --from=build --chown=65532:65532 /out/data /data
ENV ADDR=:7331 DB_PATH=/data/starbase.db
VOLUME /data
EXPOSE 7331
USER nonroot
ENTRYPOINT ["/starbase"]
