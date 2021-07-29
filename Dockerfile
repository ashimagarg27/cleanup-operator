# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /
RUN microdnf update && microdnf install -y curl bash tar gzip openssl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.6/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl &&  mv ./kubectl /usr/local/bin/kubectl

# Install OC
ARG OC_VERSION=4.6
RUN curl -sLo /tmp/oc.tar.gz https://mirror.openshift.com/pub/openshift-v$(echo $OC_VERSION | cut -d'.' -f 1)/clients/oc/$OC_VERSION/linux/oc.tar.gz && \
    tar xzvf /tmp/oc.tar.gz -C /usr/local/bin/ && \
    rm -rf /tmp/oc.tar.gz 
# CMD ["/usr/local/bin/oc"]

# # Install Tridentctl
# ARG TRIDENT_VERSION=20.07.1
# RUN curl -LO https://github.com/NetApp/trident/releases/download/v${TRIDENT_VERSION}/trident-installer-${TRIDENT_VERSION}.tar.gz && \
#     tar -xvf trident-installer-$TRIDENT_VERSION.tar.gz
# CMD ["/trident-installer/tridentctl"]

COPY --from=builder /workspace/manager .
# USER 65532:65532
USER root
ENTRYPOINT ["/manager"]
