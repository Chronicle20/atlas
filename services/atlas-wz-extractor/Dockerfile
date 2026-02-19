FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# Copy library module definitions
COPY libs/atlas-model/go.mod libs/atlas-model/go.sum libs/atlas-model/
COPY libs/atlas-rest/go.mod libs/atlas-rest/go.sum libs/atlas-rest/
COPY libs/atlas-retry/go.mod libs/atlas-retry/
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/

# Copy service module definitions
COPY services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod services/atlas-wz-extractor/atlas.com/wz-extractor/go.sum services/atlas-wz-extractor/atlas.com/wz-extractor/

# Create a minimal go.work for this build
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-model' >> go.work && \
    echo '    ./libs/atlas-rest' >> go.work && \
    echo '    ./libs/atlas-retry' >> go.work && \
    echo '    ./libs/atlas-tenant' >> go.work && \
    echo '    ./services/atlas-wz-extractor/atlas.com/wz-extractor' >> go.work && \
    echo ')' >> go.work

# Download dependencies
RUN go mod download -C services/atlas-wz-extractor/atlas.com/wz-extractor || true

# Copy library source code
COPY libs/atlas-model libs/atlas-model
COPY libs/atlas-rest libs/atlas-rest
COPY libs/atlas-retry libs/atlas-retry
COPY libs/atlas-tenant libs/atlas-tenant
# Copy service source code
COPY services/atlas-wz-extractor/atlas.com/wz-extractor services/atlas-wz-extractor/atlas.com/wz-extractor

# Build
RUN go build -C services/atlas-wz-extractor/atlas.com/wz-extractor -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
