# Stage 1: Apache Kafka CLI scripts. We need kafka-topics.sh and
# kafka-consumer-groups.sh for cleanup.sh — they ship in the upstream
# Apache Kafka tarball, not as an Alpine package.
FROM alpine:3.20 AS kafka

ARG KAFKA_VERSION=3.7.2
ARG SCALA_VERSION=2.13

RUN apk add --no-cache curl tar
RUN curl -fsSL --retry 3 --retry-delay 5 \
    "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz" \
    -o /tmp/kafka.tgz && \
    tar -xzf /tmp/kafka.tgz -C /opt && \
    mv /opt/kafka_${SCALA_VERSION}-${KAFKA_VERSION} /opt/kafka && \
    rm /tmp/kafka.tgz

# Stage 2: runtime image with all entrypoint dependencies.
FROM alpine:3.20

RUN apk add --no-cache \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        ca-certificates \
        github-cli \
        kubectl \
        openjdk17-jre-headless

# Apache Kafka CLI scripts (only what cleanup.sh needs).
COPY --from=kafka /opt/kafka/bin/kafka-topics.sh /opt/kafka/bin/kafka-topics.sh
COPY --from=kafka /opt/kafka/bin/kafka-consumer-groups.sh /opt/kafka/bin/kafka-consumer-groups.sh
COPY --from=kafka /opt/kafka/bin/kafka-run-class.sh /opt/kafka/bin/kafka-run-class.sh
COPY --from=kafka /opt/kafka/libs /opt/kafka/libs
COPY --from=kafka /opt/kafka/config /opt/kafka/config
ENV PATH="/opt/kafka/bin:${PATH}"

WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
