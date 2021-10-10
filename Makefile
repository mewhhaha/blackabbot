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
	mkdir -p build/bin
	go build -tags nolibopusfile -ldflags="-s -w" -o ./build/run ./cmd/$*
	cp /usr/bin/opusenc ./build/bin/
	cp /usr/bin/opusdec ./build/bin/
	cp /usr/bin/opusinfo ./build/bin/
	cp /usr/lib/libopus.so.0 ./build
	cp /usr/lib/libopusfile.so.0 ./build
	cd build && zip -r $*.zip ./*
	rm ./build/run
	rm -rf ./build/bin
	


deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build