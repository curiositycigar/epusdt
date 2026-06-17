SHELL := /bin/bash

.PHONY: run-local build-release deploy-remote

run-local:
	./scripts/run-local.sh

build-release:
	./scripts/build-release.sh

deploy-remote:
	./scripts/deploy-remote.sh
