FROM golang:1.11-alpine as builder
ADD . /go/src/github.com/shizhMSFT/oras
WORKDIR /go/src/github.com/shizhMSFT/oras
RUN go install github.com/shizhMSFT/oras/cmd/oras

FROM alpine
LABEL maintainer="shizh@microsoft.com"
RUN apk --update add ca-certificates
COPY --from=builder /go/bin/oras /bin/oras
RUN mkdir /workplace
WORKDIR /workplace 
ENTRYPOINT  ["/bin/oras"]
