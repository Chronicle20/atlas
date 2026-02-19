# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy library module definitions
COPY libs/atlas-constants/go.mod libs/atlas-constants/go.sum libs/atlas-constants/
COPY libs/atlas-kafka/go.mod libs/atlas-kafka/go.sum libs/atlas-kafka/
COPY libs/atlas-model/go.mod libs/atlas-model/go.sum libs/atlas-model/
COPY libs/atlas-rest/go.mod libs/atlas-rest/go.sum libs/atlas-rest/
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/

COPY libs/atlas-database/go.mod libs/atlas-database/go.sum libs/atlas-database/
COPY libs/atlas-retry/go.mod libs/atlas-retry/
# Copy service module definitions
COPY services/atlas-maps/atlas.com/maps/go.mod services/atlas-maps/atlas.com/maps/go.sum services/atlas-maps/atlas.com/maps/

# Create a minimal go.work for this build
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-constants' >> go.work && \
    echo '    ./libs/atlas-kafka' >> go.work && \
    echo '    ./libs/atlas-model' >> go.work && \
    echo '    ./libs/atlas-rest' >> go.work && \
    echo '    ./libs/atlas-tenant' >> go.work && \
    echo '    ./libs/atlas-database' >> go.work && \
    echo '    ./libs/atlas-retry' >> go.work && \
    echo '    ./services/atlas-maps/atlas.com/maps' >> go.work && \
    echo ')' >> go.work

# Download dependencies
RUN go mod download -C services/atlas-maps/atlas.com/maps || true

# Copy library source code
COPY libs/atlas-constants libs/atlas-constants
COPY libs/atlas-kafka libs/atlas-kafka
COPY libs/atlas-model libs/atlas-model
COPY libs/atlas-rest libs/atlas-rest
COPY libs/atlas-tenant libs/atlas-tenant

COPY libs/atlas-database libs/atlas-database
COPY libs/atlas-retry libs/atlas-retry
# Copy service source code
COPY services/atlas-maps/atlas.com/maps services/atlas-maps/atlas.com/maps

# Build
RUN go build -C services/atlas-maps/atlas.com/maps -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
