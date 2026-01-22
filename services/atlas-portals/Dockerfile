# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy module definitions
COPY services/atlas-portals/atlas.com/portals/go.mod services/atlas-portals/atlas.com/portals/go.sum services/atlas-portals/atlas.com/portals/

# Download dependencies
WORKDIR /app/services/atlas-portals/atlas.com/portals
RUN go mod download

# Copy source code
WORKDIR /app
COPY services/atlas-portals/atlas.com/portals services/atlas-portals/atlas.com/portals

# Build
RUN go build -C services/atlas-portals/atlas.com/portals -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
