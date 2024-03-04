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

PROJECT_PKG = oras.land/oras
CLI_EXE     = oras
CLI_PKG     = $(PROJECT_PKG)/cmd/oras
GIT_COMMIT  = $(shell git rev-parse HEAD)
GIT_TAG     = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY   = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
GO_EXE      = go

TARGET_OBJS ?= checksums.txt darwin_amd64.tar.gz darwin_arm64.tar.gz linux_amd64.tar.gz linux_arm64.tar.gz linux_armv7.tar.gz linux_s390x.tar.gz linux_ppc64le.tar.gz linux_riscv64.tar.gz windows_amd64.zip freebsd_amd64.tar.gz

LDFLAGS = -w
ifdef VERSION
	LDFLAGS += -X $(PROJECT_PKG)/internal/version.BuildMetadata=$(VERSION)
endif
ifneq ($(GIT_TAG),)
	LDFLAGS += -X $(PROJECT_PKG)/internal/version.BuildMetadata=
endif
LDFLAGS += -X $(PROJECT_PKG)/internal/version.GitCommit=${GIT_COMMIT}
LDFLAGS += -X $(PROJECT_PKG)/internal/version.GitTreeState=${GIT_DIRTY}

.PHONY: test
test: tidy vendor check-encoding  ## tidy and run tests
	$(GO_EXE) test -race -v -coverprofile=coverage.txt -covermode=atomic -coverpkg=./... ./...

.PHONY: teste2e
teste2e:  ## run end to end tests
	./test/e2e/scripts/e2e.sh $(shell git rev-parse --show-toplevel) --clean

.PHONY: covhtml
covhtml:  ## look at code coverage
	open .cover/coverage.html

.PHONY: clean
clean:  ## clean up build
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: build
build: build-linux build-mac build-windows  ## build for all targets

.PHONY: build-linux-all
build-linux-all: build-linux-amd64 build-linux-arm64 build-linux-arm-v7 build-linux-s390x build-linux-ppc64le build-linux-riscv64 ## build all linux architectures

.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64

.PHONY: build-linux-amd64
build-linux-amd64:  ## build for linux amd64
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-arm64
build-linux-arm64:  ## build for linux arm64
	GOARCH=arm64 CGO_ENABLED=0 GOOS=linux $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/arm64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-arm-v7
build-linux-arm-v7:  ## build for linux arm v7
	GOARCH=arm CGO_ENABLED=0 GOOS=linux $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/arm/v7/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-s390x
build-linux-s390x:  ## build for linux s390x
	GOARCH=s390x CGO_ENABLED=0 GOOS=linux $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/s390x/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-ppc64le
build-linux-ppc64le:  ## build for linux ppc64le
	GOARCH=ppc64le CGO_ENABLED=0 GOOS=linux $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/ppc64le/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-riscv64
build-linux-riscv64:  ## build for linux riscv64
	GOARCH=riscv64 CGO_ENABLED=0 GOOS=linux $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/riscv64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-mac
build-mac: build-mac-arm64 build-mac-amd64  ## build all mac architectures

.PHONY: build-mac-amd64
build-mac-amd64:  ## build for mac amd64
	GOARCH=amd64 CGO_ENABLED=0 GOOS=darwin $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/darwin/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-mac-arm64
build-mac-arm64:  ## build for mac arm64
	GOARCH=arm64 CGO_ENABLED=0 GOOS=darwin $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/darwin/arm64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-windows
build-windows: build-windows-amd64 build-windows-arm64  ## build all windows architectures

.PHONY: build-windows-amd64
build-windows-amd64:  ## build for windows amd64
	GOARCH=amd64 CGO_ENABLED=0 GOOS=windows $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/windows/amd64/$(CLI_EXE).exe $(CLI_PKG)

.PHONY: build-windows-arm64
build-windows-arm64:  ## build for windows arm64
	GOARCH=arm64 CGO_ENABLED=0 GOOS=windows $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/windows/arm64/$(CLI_EXE).exe $(CLI_PKG)

.PHONY: build-freebsd
build-freebsd: build-freebsd-amd64  ## build all freebsd architectures

.PHONY: build-freebsd-amd64
build-freebsd-amd64:  ## build for freebsd amd64
	GOARCH=amd64 CGO_ENABLED=0 GOOS=freebsd $(GO_EXE) build -v --ldflags="$(LDFLAGS)" \
		-o bin/freebsd/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: check-encoding
check-encoding:  ## check file CR/LF encoding
	! find cmd internal -name "*.go" -type f -exec file "{}" ";" | grep CRLF

.PHONY: fix-encoding
fix-encoding:  ## fix file CR/LF encoding
	find cmd internal -type f -name "*.go" -exec sed -i -e "s/\r//g" {} +

.PHONY: tidy
tidy:  ## go mod tidy
	GO111MODULE=on $(GO_EXE) mod tidy

.PHONY: vendor
vendor:  ## go mod vendor
	GO111MODULE=on $(GO_EXE) mod vendor

.PHONY: fetch-dist
fetch-dist:  ## fetch distribution
	mkdir -p _dist
	cd _dist && \
	for obj in ${TARGET_OBJS} ; do \
		curl -sSL -o oras_${VERSION}_$${obj} https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_$${obj} ; \
	done

.PHONY: sign
sign:  ## sign
	for f in $$(ls _dist/*.{gz,txt} 2>/dev/null) ; do \
		gpg --armor --detach-sign $${f} ; \
	done

.PHONY: teste2e-covdata
teste2e-covdata:  ## test e2e coverage
	export GOCOVERDIR=$(CURDIR)/test/e2e/.cover; \
	rm -rf $$GOCOVERDIR; \
	mkdir -p $$GOCOVERDIR; \
	$(MAKE) teste2e && $(GO_EXE) tool covdata textfmt -i=$$GOCOVERDIR -o "$(CURDIR)/test/e2e/coverage.txt"

.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[%\/0-9A-Za-z_-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
