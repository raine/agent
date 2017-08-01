FROM scratch

ARG version

COPY ./build/timber-agent-${version}-linux-amd64/ /

ENTRYPOINT ["/bin/timber-agent"]
