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
