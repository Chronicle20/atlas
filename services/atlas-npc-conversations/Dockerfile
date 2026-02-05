# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy lib and service module definitions
COPY libs/atlas-constants/go.mod libs/atlas-constants/go.sum libs/atlas-constants/
COPY libs/atlas-script-core/go.mod libs/atlas-script-core/go.sum libs/atlas-script-core/
COPY services/atlas-npc-conversations/atlas.com/npc/go.mod services/atlas-npc-conversations/atlas.com/npc/go.sum services/atlas-npc-conversations/atlas.com/npc/

# Create a minimal go.work for this build
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-constants' >> go.work && \
    echo '    ./libs/atlas-script-core' >> go.work && \
    echo '    ./services/atlas-npc-conversations/atlas.com/npc' >> go.work && \
    echo ')' >> go.work

# Download dependencies
RUN go mod download -C services/atlas-npc-conversations/atlas.com/npc

# Copy source code
COPY libs/atlas-constants libs/atlas-constants
COPY libs/atlas-script-core libs/atlas-script-core
COPY services/atlas-npc-conversations/atlas.com/npc services/atlas-npc-conversations/atlas.com/npc

# Build
RUN go build -C services/atlas-npc-conversations/atlas.com/npc -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /
COPY services/atlas-npc-conversations/conversations /conversations

CMD ["/server"]
