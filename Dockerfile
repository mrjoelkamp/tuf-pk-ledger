# Build Stage
FROM golang:1.21 AS build-stage

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

# Create appuser
RUN adduser -D -u '1001' github
USER github

COPY --from=build-stage /go/src/opkl-updater/bin/opkl-updater /opt/opkl-updater/bin/

CMD ["/opt/opkl-updater/bin/opkl-updater"]
