.PHONY: test
test:
	rm -rf .test/ && mkdir .test/
	go test -v -covermode=atomic -coverprofile=coverage.out github.com/shizhMSFT/oras/pkg/oras
	go tool cover -html=coverage.out -o=coverage.html

.PHONY: covhtml
covhtml:
	open coverage.html

.PHONY: clean
clean:
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf
