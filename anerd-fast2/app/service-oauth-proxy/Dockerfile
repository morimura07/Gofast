# Build
FROM golang:1.25-bookworm AS build
WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

# Copy go files
COPY go.mod ./
COPY *.go ./

# Download dependencies (if any in the future)
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o /oauth-proxy .

# Production
FROM gcr.io/distroless/static-debian12 AS prod
WORKDIR /

COPY --from=busybox:1.36-uclibc /bin/wget /usr/bin/wget
COPY --from=build /oauth-proxy /oauth-proxy

CMD ["/oauth-proxy"]
