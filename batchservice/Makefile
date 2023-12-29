run_linter:
	docker run --rm -it -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.55.2 golangci-lint run -v

run_tests:
	go test -v -race -short  ./... -coverprofile reports/coverage.out

show_coverage:
	go tool cover -html=reports/coverage.out

.PHONY: run_linter run_tests show_coverage
