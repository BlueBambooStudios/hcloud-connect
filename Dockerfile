FROM golang:1.14 as builder

WORKDIR /hcloud-connect/src

ADD . .

RUN CGO_ENABLED=0 go build -o hcloud-connect.bin .

FROM alpine:3.11

RUN apk add --no-cache ca-certificates

COPY --from=builder /hcloud-connect/src/hcloud-connect.bin /bin/hcloud-connect

ENTRYPOINT ["/bin/hcloud-connect"]