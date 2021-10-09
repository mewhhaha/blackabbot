dependencies:
	go mod download

build: clean build/webhook

build/%:
	docker build . \
		--tag blackabbot/$* \
		--build-arg CMD_NAME=$* \
		-f ./deployments/Dockerfile

deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build