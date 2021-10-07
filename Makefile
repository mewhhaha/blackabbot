dependencies:
	go mod download

build: clean
	mkdir -p build
	go build -o ./build/run ./cmd/main.go 
	cd build && \
		zip function.zip run
	rm ./build/run

deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build