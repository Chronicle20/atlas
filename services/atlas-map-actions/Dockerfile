FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy lib and service module definitions
COPY libs/atlas-script-core/go.mod libs/atlas-script-core/go.sum libs/atlas-script-core/
COPY services/atlas-map-actions/atlas.com/map-actions/go.mod services/atlas-map-actions/atlas.com/map-actions/go.sum* services/atlas-map-actions/atlas.com/map-actions/

# Create a minimal go.work for this build
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-script-core' >> go.work && \
    echo '    ./services/atlas-map-actions/atlas.com/map-actions' >> go.work && \
    echo ')' >> go.work

# Download dependencies
RUN go mod download -C services/atlas-map-actions/atlas.com/map-actions || true

# Copy source code
COPY libs/atlas-script-core libs/atlas-script-core
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
