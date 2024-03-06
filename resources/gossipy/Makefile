# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

VERSION=1.0

# Output helpers
# --------------

TASK_DONE = echo "✓  $@ done"
TASK_BUILD = echo "🛠️  $@ done"

export CODEBUILD_BUILD_NUMBER ?= 0
export CODEBUILD_RESOLVED_SOURCE_VERSION ?=$(shell git rev-list -1 HEAD --abbrev-commit)
export BUILD_DATE=$(shell date -u '+%Y%m%d')

all: test gossipy/build

deps:
	go get -v  ./...
	
gossipy/build:
	$(GOBUILD) -ldflags " \
		-X gossipy/pkg/version.Version=${VERSION}.${CODEBUILD_BUILD_NUMBER} \
		-X gossipy/pkg/version.BuildHash=${CODEBUILD_RESOLVED_SOURCE_VERSION} \
		-X gossipy/pkg/version.BuildDate=${BUILD_DATE}" \
		-o ./gossipy -v ./cmd
	@$(TASK_BUILD)

test: 
	@$(GOTEST) -v ./...
	@$(TASK_DONE)

clean:
	@$(GOCLEAN)
	@rm -f ./gossipy
	@$(TASK_DONE)