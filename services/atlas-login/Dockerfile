# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy library module definitions
COPY libs/atlas-constants/go.mod libs/atlas-constants/go.sum libs/atlas-constants/
COPY libs/atlas-kafka/go.mod libs/atlas-kafka/go.sum libs/atlas-kafka/
COPY libs/atlas-model/go.mod libs/atlas-model/go.sum libs/atlas-model/
COPY libs/atlas-packet/go.mod libs/atlas-packet/go.sum libs/atlas-packet/
COPY libs/atlas-rest/go.mod libs/atlas-rest/go.sum libs/atlas-rest/
COPY libs/atlas-socket/go.mod libs/atlas-socket/go.sum libs/atlas-socket/
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/

COPY libs/atlas-opcodes/go.mod libs/atlas-opcodes/
COPY libs/atlas-retry/go.mod libs/atlas-retry/
COPY libs/atlas-service/go.mod libs/atlas-service/
# Copy service module definitions
COPY services/atlas-login/atlas.com/login/go.mod services/atlas-login/atlas.com/login/go.sum services/atlas-login/atlas.com/login/

# Create a minimal go.work for this build
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-constants' >> go.work && \
    echo '    ./libs/atlas-kafka' >> go.work && \
    echo '    ./libs/atlas-model' >> go.work && \
    echo '    ./libs/atlas-packet' >> go.work && \
    echo '    ./libs/atlas-rest' >> go.work && \
    echo '    ./libs/atlas-socket' >> go.work && \
    echo '    ./libs/atlas-tenant' >> go.work && \
    echo '    ./libs/atlas-opcodes' >> go.work && \
    echo '    ./libs/atlas-retry' >> go.work && \
    echo '    ./libs/atlas-service' >> go.work && \
    echo '    ./services/atlas-login/atlas.com/login' >> go.work && \
    echo ')' >> go.work

# Download dependencies
RUN go mod download -C services/atlas-login/atlas.com/login || true

# Copy library source code
COPY libs/atlas-constants libs/atlas-constants
COPY libs/atlas-kafka libs/atlas-kafka
COPY libs/atlas-model libs/atlas-model
COPY libs/atlas-packet libs/atlas-packet
COPY libs/atlas-rest libs/atlas-rest
COPY libs/atlas-socket libs/atlas-socket
COPY libs/atlas-tenant libs/atlas-tenant

COPY libs/atlas-opcodes libs/atlas-opcodes
COPY libs/atlas-retry libs/atlas-retry
COPY libs/atlas-service libs/atlas-service
# Copy service source code
COPY services/atlas-login/atlas.com/login services/atlas-login/atlas.com/login

# Force local resolution of Chronicle20/atlas-* modules (bypasses unreachable github.com publishes).
RUN cd services/atlas-login/atlas.com/login && \
    go mod edit \
      -replace=github.com/Chronicle20/atlas/libs/atlas-constants=/app/libs/atlas-constants \
      -replace=github.com/Chronicle20/atlas/libs/atlas-kafka=/app/libs/atlas-kafka \
      -replace=github.com/Chronicle20/atlas/libs/atlas-model=/app/libs/atlas-model \
      -replace=github.com/Chronicle20/atlas/libs/atlas-opcodes=/app/libs/atlas-opcodes \
      -replace=github.com/Chronicle20/atlas/libs/atlas-packet=/app/libs/atlas-packet \
      -replace=github.com/Chronicle20/atlas/libs/atlas-rest=/app/libs/atlas-rest \
      -replace=github.com/Chronicle20/atlas/libs/atlas-retry=/app/libs/atlas-retry \
      -replace=github.com/Chronicle20/atlas/libs/atlas-service=/app/libs/atlas-service \
      -replace=github.com/Chronicle20/atlas/libs/atlas-socket=/app/libs/atlas-socket \
      -replace=github.com/Chronicle20/atlas/libs/atlas-tenant=/app/libs/atlas-tenant \
    && go mod tidy

# Build
RUN go build -C services/atlas-login/atlas.com/login -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
