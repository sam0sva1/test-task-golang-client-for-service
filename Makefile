ROOT=$(shell pwd)

run_dev_api:
	go run cmd/app/main.go

run_linter:
	docker run --rm \
		-v $(GOPATH)/pkg/mod:/go/pkg/mod \
		-v $(ROOT):/cmd/app \
		-w /app \
		golangci/golangci-lint:v1.55.2 golangci-lint run -v

run_tests:
	go test -v -race -short  ./... -coverprofile coverage.out

show_coverage:
	go tool cover -html=coverage.out