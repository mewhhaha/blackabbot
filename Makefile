ECR_REPOSITORY ?= blackabbot
ECR_TAG ?= latest


dependencies:
	go mod download

build: clean build/webhook

build/%:
	docker build . \
		--tag ${ECR_REPOSITORY}:${ECR_TAG} \
		--build-arg CMD_NAME=$* \
		-f ./deployments/Dockerfile

push:
	docker push --all-tags ${ECR_REPOSITORY}

deploy:
	set -e
	cd terraform \
		&& terraform init \
		&& terraform validate \
		&& terraform apply --auto-approve

clean:
	rm -rf build