FROM alpine:3

ARG TARGETOS
ARG TARGETARCH

LABEL org.opencontainers.image.title="webhook Stela" \
  org.opencontainers.image.description="Send build notifications to Stela via webhook" \
  org.opencontainers.image.vendor="ktmbees"
LABEL org.opencontainers.image.source=https://github.com/devopsprabin/stela-webhook

RUN apk add --no-cache ca-certificates && \
  rm -rf /var/cache/apk/*

COPY release/${TARGETOS}/${TARGETARCH}/stela-webhook /bin/

ENTRYPOINT ["/bin/stela-webhook"]
