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
	go build -o ./build/$*/run ./cmd/$*

	cp /usr/lib/x86_64-linux-gnu/libopus.so.0 ./build/$*/
	cp /usr/lib/libopusfile.so.0 ./build/$*/
	cp /usr/lib/i386-linux-gnu/libogg.so.0 ./build/$*/

	cd build/$* && patchelf --set-rpath "$$ORIGIN" run
	cd build/$* && zip -r function.zip ./*

	rm ./build/$*/run
	rm ./build/$*/libopusfile.so.0
	rm ./build/$*/libopus.so.0
	rm ./build/$*/libogg.so.0
	

deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build