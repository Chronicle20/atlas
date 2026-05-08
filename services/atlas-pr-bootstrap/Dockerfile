FROM alpine:3.20

RUN apk add --no-cache \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        kafka-tools \
        ca-certificates \
        github-cli \
        kubectl

WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
