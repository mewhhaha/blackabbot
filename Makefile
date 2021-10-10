CURRENT_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(dir $(CURRENT_PATH))

.PHONY: dependencies docker/build build build/% deploy clean

dependencies:
	go mod download

docker/build: 
	docker build . -t blackabbot/builder -f ./tools/Dockerfile
	docker run --rm -v ${CURRENT_DIR}:/project blackabbot/builder 

build: clean build/webhook

build/%:
	mkdir -p build/$*
	go build -ldflags="-s -w" -o ./build/run ./cmd/$*
	cd build && zip -r $*/function.zip ./*
	rm ./build/run
	


deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build