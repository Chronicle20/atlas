FROM golang:1.25.5-alpine3.21 AS build-env

RUN apk add --no-cache git

WORKDIR /app

COPY libs/atlas-constants/go.mod libs/atlas-constants/go.sum libs/atlas-constants/
COPY libs/atlas-database/go.mod libs/atlas-database/go.sum libs/atlas-database/
COPY libs/atlas-kafka/go.mod libs/atlas-kafka/go.sum libs/atlas-kafka/
COPY libs/atlas-model/go.mod libs/atlas-model/go.sum libs/atlas-model/
COPY libs/atlas-redis/go.mod libs/atlas-redis/go.sum libs/atlas-redis/
COPY libs/atlas-rest/go.mod libs/atlas-rest/go.sum libs/atlas-rest/
COPY libs/atlas-tenant/go.mod libs/atlas-tenant/go.sum libs/atlas-tenant/

COPY services/atlas-merchant/atlas.com/merchant/go.mod services/atlas-merchant/atlas.com/merchant/go.sum services/atlas-merchant/atlas.com/merchant/

RUN echo 'go 1.25.5' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./libs/atlas-constants' >> go.work && \
    echo '    ./libs/atlas-database' >> go.work && \
    echo '    ./libs/atlas-kafka' >> go.work && \
    echo '    ./libs/atlas-model' >> go.work && \
    echo '    ./libs/atlas-redis' >> go.work && \
    echo '    ./libs/atlas-rest' >> go.work && \
    echo '    ./libs/atlas-tenant' >> go.work && \
    echo '    ./services/atlas-merchant/atlas.com/merchant' >> go.work && \
    echo ')' >> go.work

RUN go mod download -C services/atlas-merchant/atlas.com/merchant || true

COPY libs/atlas-constants libs/atlas-constants
COPY libs/atlas-database libs/atlas-database
COPY libs/atlas-kafka libs/atlas-kafka
COPY libs/atlas-model libs/atlas-model
COPY libs/atlas-redis libs/atlas-redis
COPY libs/atlas-rest libs/atlas-rest
COPY libs/atlas-tenant libs/atlas-tenant

COPY services/atlas-merchant/atlas.com/merchant services/atlas-merchant/atlas.com/merchant

RUN go build -C services/atlas-merchant/atlas.com/merchant -o /server

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /

CMD ["/server"]
