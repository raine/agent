FROM alpine:3.6

ARG version

RUN mkdir -p /opt
WORKDIR /opt

COPY ./build/timber-agent-${version}-linux-amd64/ /opt/timber-agent/

ENTRYPOINT ["/opt/timber-agent/bin/timber-agent"]
