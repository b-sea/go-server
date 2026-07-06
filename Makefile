GOLANGCILINT_VERSION=v2.6.1
GOLANGCILINT_ROOT=$$(go env GOPATH)/golangci-lint
GOLANGCILINT_PATH=${GOLANGCILINT_ROOT}/${GOLANGCILINT_VERSION}

.PHONY: tidy test setup-coverage coverage setup-lint lint gqlgen clean

tidy:
	go mod tidy

test:
	go test -race ./... -coverprofile=./tools/cover.out -covermode=atomic ./...

setup-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

coverage: setup-coverage test
	$$(go env GOPATH)/bin/go-test-coverage --config=./tools/.testcoverage.yml

setup-lint:
	@mkdir -p ${GOLANGCILINT_ROOT}
	@mkdir -p ${GOLANGCILINT_PATH}
	@if [ -z "$$(ls -A "${GOLANGCILINT_PATH}")" ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOLANGCILINT_PATH}" "${GOLANGCILINT_VERSION}"; \
	fi

lint: setup-lint
	@${GOLANGCILINT_PATH}/golangci-lint cache clean
	@${GOLANGCILINT_PATH}/golangci-lint run -c tools/.golangci.yml

openapi:
	@go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen \
		--config=tools/oapi-codegen.yaml \
		api/api.yml