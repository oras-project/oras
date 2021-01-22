FROM docker.io/library/golang:1.15.7-alpine as builder
RUN apk add git make
ENV ORASPKG /oras
ADD . ${ORASPKG}
WORKDIR ${ORASPKG}
RUN make build-linux
RUN mv ${ORASPKG}/bin/linux/amd64/oras /go/bin/oras

FROM docker.io/library/alpine:3.13.0
LABEL maintainer="shizh@microsoft.com"
RUN apk --update add ca-certificates
COPY --from=builder /go/bin/oras /bin/oras
RUN mkdir /workspace
WORKDIR /workspace
ENTRYPOINT  ["/bin/oras"]
