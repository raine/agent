# The Mozilla CA Certificate Store is being redistributed within this 
# container under the Mozilla Public License 2.0 via the Alpine Linux 
# ca-certificates package.
# https://www.mozilla.org/en-US/about/governance/policies/security-group/certs/
# https://www.mozilla.org/media/MPL/2.0/index.815ca599c9df.txt
# https://pkgs.alpinelinux.org/package/edge/main/x86_64/ca-certificates

FROM alpine:3.7 as certs

RUN apk add --no-cache ca-certificates


FROM scratch

ARG version

COPY --from=certs /etc/ssl/certs/ca-certificates.crt \ 
     /etc/ssl/certs/ca-certificates.crt
COPY ./build/timber-agent-${version}-linux-amd64/timber-agent /timber-agent

ENTRYPOINT ["/timber-agent/bin/timber-agent"]
