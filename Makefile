STABLE_VERSION := $(shell cat util/stable_version.txt)
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

platform=$(shell uname -o)

ifeq ($(OS),Windows_NT)
	PATH_SEPARATOR:=;
else
	PATH_SEPARATOR:=:
endif
# Add executable to PATH so it can be used as the git sequence editor for unit tests.
export PATH := ${PATH}${PATH_SEPARATOR}${ROOT_DIR}/bin

.PHONY: setup
setup:
	git config core.hooksPath .githooks

.PHONY: format
format:
	go fmt ./...
ifeq ($(platform),Darwin) # Mac
	sed -i '' -e 's/gh-stacked-diff\/v2@v2\.[0-9]*\.[0-9]*/gh-stacked-diff\/v2@v'${STABLE_VERSION}'/' README.md
else # Windows
	sed -i 's/gh-stacked-diff\\/v2@v2\.[0-9]*\.[0-9]*/gh-stacked-diff\\/v2@v'${STABLE_VERSION}'/' README.md
endif
.PHONY: build
build: format
	mkdir -p bin
	go build -o bin

.PHONY: lint
lint: build
	golangci-lint run

.PHONY: update-readme
update-readme: build
	./update-readme-help.sh

# Example TEST_ARGS:
# make TEST_ARGS="-timeout 10s -run TestSdUpdate_WhenDestinationCommitNotSpecified_UpdatesSelectedPr" -o lint test
# Note: timeout is cumulative for all tests to run.
.PHONY: test
test: build lint
	go test -v -timeout 120s ${TEST_ARGS} ./...
