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
	CGO_ENABLED=0 go test -v -covermode=atomic -coverprofile=coverage.out github.com/shizhMSFT/oras/pkg/oras
	go tool cover -html=coverage.out -o=coverage.html

.PHONY: covhtml
covhtml:
	open coverage.html

.PHONY: clean
clean:
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf
