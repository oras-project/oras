CLI_EXE=oras
CLI_PKG=github.com/deislabs/oras/cmd/oras

.PHONY: update-deps
update-deps:
	dep ensure --update

# Note: The dependency "rsc.io/letsencrypt" uses a vanity URL, so must run
# the wget command to get the correct version
.PHONY: fix-deps
fix-deps:
	wget -O vendor/rsc.io/letsencrypt/lets.go https://raw.githubusercontent.com/dmcgowan/letsencrypt/e770c10b0f1a64775ae91d240407ce00d1a5bdeb/lets.go

.PHONY: test
test:
	./scripts/test.sh

.PHONY: covhtml
covhtml:
	open .cover/coverage.html

.PHONY: clean
clean:
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: build
build: build-linux build-mac build-windows

.PHONY: build-linux
build-linux:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -v --ldflags="-w" \
		-o bin/linux/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-mac
build-mac:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=darwin go build -v --ldflags="-w" \
		-o bin/darwin/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-windows
build-windows:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=windows go build -v --ldflags="-w" \
		-o bin/windows/amd64/$(CLI_EXE).exe $(CLI_PKG)
