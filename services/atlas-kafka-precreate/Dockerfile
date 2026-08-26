# syntax=docker/dockerfile:1.26
# Sync-wave-0 Kafka topic pre-creation tool. Replaces the apache/kafka:3.7.2
# image and its JVM CLI (deploy/k8s/base/kafka-precreate.sh, deleted in
# task-260): the entire topic pass is one CreateTopics request over one
# connection instead of ~170 JVM cold starts.
#
# The runtime stage is scratch on purpose (NFR-9): no shell, no Kafka CLI,
# no JRE. The only thing copied in beside the binary is the CA bundle.
ARG GO_VERSION=1.27.0
ARG ALPINE_VERSION=3.24

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /atlas-kafka-precreate .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /atlas-kafka-precreate /atlas-kafka-precreate
USER 65532:65532
ENTRYPOINT ["/atlas-kafka-precreate"]
