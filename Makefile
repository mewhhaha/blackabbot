dependencies:
	go mod download

build: clean
	mkdir -p build
	go build -o ./build/run ./cmd/main.go 
	zip build/function.zip build/run
	rm ./build/run

deploy:
	set -e
	cd terraform \
		&& terraform init
		&& terraform validate
		&& terraform apply	

clean:
	rm -rf build