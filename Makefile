
# Image URL to use all building/pushing image targets
IMG ?= carolynvs/cloudkinds-servicecatalog
TAG ?= latest

all: test provider

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build provider binary
provider: generate fmt vet
	go build -o bin/manager github.com/carolynvs/cloudkinds/cmd/provider

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: docker-push
	helm upgrade --install cloudkinds-svcat charts/cloudkinds-servicecatalog \
	  --recreate-pods --set imagePullPolicy="Always",deploymentStrategy="Recreate"

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Build the docker image
docker-build:
	docker build -t ${IMG}:${TAG} -f cmd/manager/Dockerfile .
	docker build -t ${IMG}-sampleprovider:${TAG} -f cmd/sampleprovider/Dockerfile .
	docker build -t ${IMG}-servicecatalog:${TAG} -f cmd/servicecatalog/Dockerfile .
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push: docker-build
	docker push ${IMG}-servicecatalog:${TAG}
