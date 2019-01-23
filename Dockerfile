FROM golang:1.11-alpine as builder
ADD . /go/src/github.com/deislabs/oras
WORKDIR /go/src/github.com/deislabs/oras
RUN go install github.com/deislabs/oras/cmd/oras

FROM alpine
LABEL maintainer="shizh@microsoft.com"
RUN apk --update add ca-certificates
COPY --from=builder /go/bin/oras /bin/oras
RUN mkdir /workspace
WORKDIR /workspace
ENTRYPOINT  ["/bin/oras"]
