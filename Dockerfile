# Build Stage
FROM golang:1.21 AS build-stage

LABEL app="build-opkl-updater"
LABEL REPO="https://github.com/mrjoelkamp/opkl-updater"

ENV PROJPATH=/go/src/github.com/mrjoelkamp/opkl-updater

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

ADD . /go/src/github.com/mrjoelkamp/opkl-updater
WORKDIR /go/src/github.com/mrjoelkamp/opkl-updater

RUN make build-golang

# Final Stage
FROM golang:1.21

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/mrjoelkamp/opkl-updater"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:/opt/opkl-updater/bin

WORKDIR /opt/opkl-updater/bin

COPY --from=build-stage /go/src/github.com/mrjoelkamp/opkl-updater/bin/opkl-updater /opt/opkl-updater/bin/
RUN chmod +x /opt/opkl-updater/bin/opkl-updater

# Create appuser
RUN adduser -D -g '' opkl-updater
USER opkl-updater

ENTRYPOINT ["/opt/opkl-updater/bin/opkl-updater"]
