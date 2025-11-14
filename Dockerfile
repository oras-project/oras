# Copyright The ORAS Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.25.4-alpine as builder
ARG TARGETPLATFORM
RUN apk add git make
ENV ORASPKG /oras
ADD . ${ORASPKG}
WORKDIR ${ORASPKG}
RUN make "build-$(echo $TARGETPLATFORM | tr / -)"
RUN mv ${ORASPKG}/bin/${TARGETPLATFORM}/oras /go/bin/oras

FROM docker.io/library/alpine:3.22.2
RUN apk --update add ca-certificates
COPY --from=builder /go/bin/oras /bin/oras
RUN mkdir /workspace
WORKDIR /workspace
ENTRYPOINT  ["/bin/oras"]
