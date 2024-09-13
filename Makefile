# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

# Default number of clusters
N ?= 3


# Build all etcds
all: test build

# Build etcd directories and binaries
build:
	mkdir -p bin/etcd/; \
	rm -rf bin/etcd/etcd; \
	$(GOBUILD) -o bin/etcd/etcd ./src/cmd/etcd/main.go; \

# Run all etcds
run:
	for i in $(shell seq 1 $(N)); do \
		echo "Running etcd$$i..."; \
		(cd bin/etcd/ && ./etcd -c ../../config/etcd$$i/config.yml) & \
	done

stop:
	pkill -f ./etcd
