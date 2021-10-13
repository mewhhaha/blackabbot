CURRENT_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(dir $(CURRENT_PATH))

.PHONY: dependencies build build/% deploy clean

dependencies:
	go mod download

build: clean build/webhook build/reply

build/%:
	docker build . \
		--tag $* \
		--build-arg CMD_NAME=$* \
		-f  ./tools/Dockerfile

	

deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build