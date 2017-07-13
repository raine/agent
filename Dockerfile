FROM alpine:3.6

RUN mkdir -p /opt
WORKDIR /opt
COPY ./build/amd64-linux/ /opt

ENTRYPOINT ["/opt/timber-agent/bin/timber-agent"]
