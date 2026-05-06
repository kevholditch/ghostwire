FROM alpine:3.20

RUN apk add --no-cache iproute2 iputils-ping wireguard-tools

COPY e2e/bin/ghostwire-control /usr/local/bin/ghostwire-control
COPY e2e/bin/ghostwire-agent /usr/local/bin/ghostwire-agent
