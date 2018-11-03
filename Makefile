
# Image URL to use all building/pushing image targets
REGISTRY ?= carolynvs/
IMG ?= ${REGISTRY}cloudkinds-servicecatalog
TAG ?= latest

all: test build

# Run tests
test: build fmt vet
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build provider binary
build:
	go build -o bin/servicecatalog github.com/carolynvs/cloudkinds-servicecatalog/cmd/servicecatalog

# Run against the configured Kubernetes cluster in ~/.kube/config
run:
	go run ./cmd/servicecatalog/main.go

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy:
	helm upgrade --install cloudkinds-servicecatalog charts/cloudkinds-servicecatalog \
		--recreate-pods \
		--set image.registry="${IMG}",image.tag="${TAG}" \
   		--set imagePullPolicy="Always",deploymentStrategy="Recreate"

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Build the docker image
docker-build:
	docker build -t ${IMG}:${TAG} -f cmd/servicecatalog/Dockerfile .

# Push the docker image
docker-push: docker-build
	docker push ${IMG}:${TAG}
