# Dockerfile for building the containerized poller_exporter
# golang:1.18 as of 2022-07-04
FROM golang@sha256:1bbb02af44e5324a6eabe502b6a928d368977225c0255bc9aca4a734145f86e1 AS build

COPY ./ /workdir/
WORKDIR /workdir

RUN mkdir -p /config/app /config/badges

RUN go run mage.go binary

FROM scratch

MAINTAINER Will Rouesnel <wrouesnel@wrouesnel.com>
EXPOSE 8080

ENV PATH=/bin
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs
COPY --from=build /workdir/badgeserv /bin/badgeserv

# Ensure the container configuration is usable
COPY --from=build /config /config

ENV BADGESERV_CONFIGFILE=/config/app/badgeserv.yml

ENTRYPOINT ["/bin/badgeserv"]
CMD ["--logging.format=json", "api", "--badge-config-dir=/config/badges"]
