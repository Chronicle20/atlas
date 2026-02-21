FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy library module definitions
COPY libs/atlas-constants/go.mod libs/atlas-constants/go.sum libs/atlas-constants/
COPY libs/atlas-database/go.mod libs/atlas-database/go.sum libs/atlas-database/
COPY libs/atlas-kafka/go.mod libs/atlas-kafka/go.sum libs/atlas-kafka/
COPY libs/atlas-model/go.mod libs/atlas-model/go.sum libs/atlas-model/
COPY libs/atlas-rest/go.mod libs/atlas-rest/go.sum libs/atlas-rest/
COPY libs/atlas-script-core/go.mod libs/atlas-script-core/go.sum libs/atlas-script-core/
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/

COPY libs/atlas-retry/go.mod libs/atlas-retry/
COPY libs/atlas-service/go.mod libs/atlas-service/
COPY libs/atlas-saga/go.mod libs/atlas-saga/
# Copy service module definitions
COPY services/atlas-map-actions/atlas.com/map-actions/go.mod services/atlas-map-actions/atlas.com/map-actions/go.sum* services/atlas-map-actions/atlas.com/map-actions/

# Create a minimal go.work for this build
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-constants' >> go.work && \
    echo '    ./libs/atlas-database' >> go.work && \
    echo '    ./libs/atlas-kafka' >> go.work && \
    echo '    ./libs/atlas-model' >> go.work && \
    echo '    ./libs/atlas-rest' >> go.work && \
    echo '    ./libs/atlas-script-core' >> go.work && \
    echo '    ./libs/atlas-tenant' >> go.work && \
    echo '    ./libs/atlas-retry' >> go.work && \
    echo '    ./libs/atlas-service' >> go.work && \
    echo '    ./services/atlas-map-actions/atlas.com/map-actions' >> go.work && \
    echo '    ./libs/atlas-saga' >> go.work && \
    echo ')' >> go.work

# Download dependencies
RUN go mod download -C services/atlas-map-actions/atlas.com/map-actions || true

# Copy library source code
COPY libs/atlas-constants libs/atlas-constants
COPY libs/atlas-database libs/atlas-database
COPY libs/atlas-kafka libs/atlas-kafka
COPY libs/atlas-model libs/atlas-model
COPY libs/atlas-rest libs/atlas-rest
COPY libs/atlas-script-core libs/atlas-script-core
COPY libs/atlas-tenant libs/atlas-tenant

COPY libs/atlas-retry libs/atlas-retry
COPY libs/atlas-service libs/atlas-service
COPY libs/atlas-saga libs/atlas-saga
# Copy service source code
COPY services/atlas-map-actions/atlas.com/map-actions services/atlas-map-actions/atlas.com/map-actions

# Build
RUN go build -C services/atlas-map-actions/atlas.com/map-actions -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /
COPY services/atlas-map-actions/scripts /scripts

CMD ["/server"]
