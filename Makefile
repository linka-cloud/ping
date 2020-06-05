

.PHONY: tests
tests:
	@docker run --rm -i -t -v $(PWD):/ping -w /ping golang:1.14 go test -race -v
