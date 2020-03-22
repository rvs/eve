FROM golang:1.13-alpine AS build-env

WORKDIR /go/src/tailscale

COPY tailscale/go.mod .
COPY tailscale/go.sum .
RUN go mod download

COPY tailscale /go/src/tailscale

RUN go install -v ./cmd/...
RUN apk add --no-cache binutils && strip /go/bin/*
RUN apk add --no-cache openssh-keygen && mkdir /etc/ssh && ssh-keygen -A

FROM alpine:3.8
ENTRYPOINT []
WORKDIR /

# RUN apk add --no-cache pciutils usbutils vim tcpdump gdb util-linux
RUN apk add --no-cache ca-certificates iptables bash
COPY --from=build-env /go/bin/* /bin/
COPY --from=build-env /etc/ssh /etc/ssh

COPY debug-spin.sh /

CMD ["/debug-spin.sh"]
