FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

# (1) go.mod COPYs
# libs/atlas-wz has no go.sum (single dep on golang.org/x/image is hosted at
# its toolchain default); follow Task 7's pattern of copying just go.mod.
COPY libs/atlas-model/go.mod libs/atlas-model/go.sum libs/atlas-model/
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/
COPY libs/atlas-wz/go.mod libs/atlas-wz/
COPY services/atlas-renders/atlas.com/renders/go.mod services/atlas-renders/atlas.com/renders/go.sum services/atlas-renders/atlas.com/renders/

# (2) Synthesized go.work
RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-model' >> go.work && \
    echo '    ./libs/atlas-tenant' >> go.work && \
    echo '    ./libs/atlas-wz' >> go.work && \
    echo '    ./services/atlas-renders/atlas.com/renders' >> go.work && \
    echo ')' >> go.work

RUN go mod download -C services/atlas-renders/atlas.com/renders || true

# (3) Source COPYs
COPY libs/atlas-model libs/atlas-model
COPY libs/atlas-tenant libs/atlas-tenant
COPY libs/atlas-wz libs/atlas-wz
COPY services/atlas-renders/atlas.com/renders services/atlas-renders/atlas.com/renders

# (4) go mod edit -replace
RUN cd services/atlas-renders/atlas.com/renders && \
    go mod edit \
      -replace=github.com/Chronicle20/atlas/libs/atlas-model=/app/libs/atlas-model \
      -replace=github.com/Chronicle20/atlas/libs/atlas-tenant=/app/libs/atlas-tenant \
      -replace=github.com/Chronicle20/atlas/libs/atlas-wz=/app/libs/atlas-wz \
    && go mod tidy

RUN go build -C services/atlas-renders/atlas.com/renders -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /server

ENTRYPOINT ["/server"]
