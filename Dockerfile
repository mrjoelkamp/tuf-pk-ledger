# Build Stage
FROM golang:1.21-alpine AS build-stage

LABEL app="build-opkl-updater"
LABEL REPO="https://github.com/mrjoelkamp/opkl-updater"

ENV PROJPATH=/go/src/opkl-updater

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

ADD . /go/src/opkl-updater
WORKDIR /go/src/opkl-updater

RUN make build-alpine

# Final Stage
FROM alpine:latest

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/mrjoelkamp/opkl-updater"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:/opt/opkl-updater/bin

WORKDIR /opt/opkl-updater/bin

COPY --from=build-stage /go/src/opkl-updater/bin/opkl-updater \
/go/src/opkl-updater/config.opkl-updater.yaml /opt/opkl-updater/bin/
RUN chmod +x /opt/opkl-updater/bin/opkl-updater

# Create appuser
RUN adduser -D -g '1000' opkl-updater
USER opkl-updater

CMD ["/opt/opkl-updater/bin/opkl-updater"]
