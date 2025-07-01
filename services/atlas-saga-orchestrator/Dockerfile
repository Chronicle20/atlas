# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.24.4-alpine3.21 AS build-env

# Copy the local package files to the container's workspace.

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN apk add --no-cache git

ADD ./atlas.com/saga-orchestrator/go.mod ./atlas.com/saga-orchestrator/go.sum /atlas.com/saga-orchestrator/
WORKDIR /atlas.com/saga-orchestrator
RUN go mod download

ADD ./atlas.com/saga-orchestrator /atlas.com/saga-orchestrator
RUN go build -o /server

FROM alpine:3.22

# Port 8080 belongs to our application
EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
