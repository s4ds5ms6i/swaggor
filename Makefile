TEST_PROJECT_ROOT_DIR="/home/shahram/projects/minimal-user-profile"
DEFAULT_OUTPUT="./swagger.yml"
DEFAULT_EXCLUDED_DIRS="vendor,docs,assets"

############################################################
# Build & Run
############################################################
build:
	go build -v -race .

run:
	go run main.go generate -r $(TEST_PROJECT_ROOT_DIR) -e $(DEFAULT_EXCLUDED_DIRS) -o $(DEFAULT_OUTPUT)

vendor:
	go mod vendor -v

############################################################
# Test & Coverage
############################################################
check-gotestsum:
	which gotestsum || (go get -u gotest.tools/gotestsum)

test: check-gotestsum vendor
	gotestsum --junitfile-testcase-classname short --junitfile .report.xml -- -gcflags 'all=-N -l' -mod vendor ./...

coverage: vendor
	gotestsum -- -gcflags 'all=-N -l' -mod vendor -v -coverprofile=.testCoverage.txt ./...
	GOFLAGS=-mod=vendor go tool cover -func=.testCoverage.txt

coverage-report: coverage
	GOFLAGS=-mod=vendor go tool cover -html=.testCoverage.txt -o testCoverageReport.html

############################################################
# Format & Lint
############################################################
check-goimport:
	which goimports || GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports

format: check-goimport
	find $(ROOT) -type f -name "*.go" -not -path "$(ROOT)/vendor/*" | xargs -n 1 -I R goimports -w R
	find $(ROOT) -type f -name "*.go" -not -path "$(ROOT)/vendor/*" | xargs -n 1 -I R gofmt -s -w R

check-golint:
	which golint || (go get -u golang.org/x/lint/golint)

lint: check-golint
	find $(ROOT) -type f -name "*.go" -not -path "$(ROOT)/vendor/*" | xargs -n 1 -I R golint -set_exit_status R

check-golangci-lint:
	which golangci-lint || (go get -u github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2)

lint-ci: check-golangci-lint vendor
	golangci-lint run -c .golangci.yml ./... --timeout=$(LINT_TIMEOUT)