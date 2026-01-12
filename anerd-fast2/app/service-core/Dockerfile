# Development
FROM golang:1.25-bookworm AS dev
WORKDIR /app

# Install Go dependencies
RUN go install github.com/air-verse/air@latest

# Copy all module and workspace files to leverage Docker cache
COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY ./gen ./gen
COPY ./service-core/go.mod ./service-core/go.sum ./service-core/
COPY ./public.pem /public.pem
COPY ./private.pem /private.pem

# Download Go modules
RUN go work sync
RUN cd service-core && go mod download

COPY . .

WORKDIR ./service-core

CMD ["air", "-c", ".air.toml"]

# Build
FROM golang:1.25-bookworm AS build
WORKDIR /

ARG TARGETOS
ARG TARGETARCH

COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY ./service-core/go.mod ./service-core/
COPY ./service-core/go.sum ./service-core/
COPY ./gen ./gen
RUN go work sync
RUN cd service-core && go mod download

COPY . .

RUN --mount=type=secret,id=privatekey cat /run/secrets/privatekey > /private.pem
RUN --mount=type=secret,id=publickey cat /run/secrets/publickey > /public.pem

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o /app ./service-core

# Production
FROM gcr.io/distroless/base-debian12 AS prod
WORKDIR /

COPY --from=busybox:1.36-uclibc /bin/wget /usr/bin/wget
COPY --from=build /app /app
COPY --from=build /public.pem /public.pem
COPY --from=build /private.pem /private.pem

CMD ["/app"]
