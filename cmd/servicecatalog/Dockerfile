# Build the provider binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/carolynvs/cloudkinds-servicecatalog
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o servicecatalog github.com/carolynvs/cloudkinds-servicecatalog/cmd/servicecatalog

# Copy the provider into a thin image
FROM ubuntu:latest
EXPOSE 8080
WORKDIR /root/
COPY --from=builder /go/src/github.com/carolynvs/cloudkinds-servicecatalog/servicecatalog .
ENTRYPOINT ["./servicecatalog"]
