FROM alpine:3

ARG TARGETOS
ARG TARGETARCH

LABEL org.opencontainers.image.title="Stela Action" \
  org.opencontainers.image.description="Send build notifications to Stela via webhook" \
  org.opencontainers.image.vendor="ktmbees"
LABEL org.opencontainers.image.source=https://github.com/devopsprabin/stela-action

# Upgrade base packages so security fixes land without waiting for a new Alpine tag.
RUN apk upgrade --no-cache && \
  apk add --no-cache ca-certificates && \
  rm -rf /var/cache/apk/*

COPY release/${TARGETOS}/${TARGETARCH}/stela-action /bin/

ENTRYPOINT ["/bin/stela-action"]
