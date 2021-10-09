dependencies:
	go mod download

build: clean build/webhook

build/%:
	mkdir -p build
	go build -o ./build/run ./cmd/$*/main.go 
	cd build && \
		zip $*.zip run
	rm ./build/run

deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build