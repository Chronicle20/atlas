# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.25.1-alpine3.21 AS build-env

# Copy the local package files to the container's workspace.

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN apk add --no-cache git

ADD ./atlas.com/atlas-compartment-transfer/go.mod ./atlas.com/atlas-compartment-transfer/go.sum /atlas.com/atlas-compartment-transfer/
WORKDIR /atlas.com/atlas-compartment-transfer
RUN go mod download

ADD ./atlas.com/atlas-compartment-transfer /atlas.com/atlas-compartment-transfer
RUN go build -o /server

FROM alpine:3.22

# Port 8080 belongs to our application
EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
