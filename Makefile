BIN_DIR = bin
TEST_DIRS = $(shell dirname `git ls-files '*_test.go'`)
TOOL_DIRS = $(shell dirname `git ls-files 'tools/*/main.go'`)
MAIN_DIRS = $(shell dirname `git ls-files '*/main.go'`)

all: deps protos vpservice revtr atlas plcontroller controller plvp

deps:
	go get -d -v github.com/NEU-SNS/ReverseTraceroute/...

protos:
	@ if ! command -v protoc >/dev/null; then \
		echo "protoc must be installed" >&2;\
		exit 1; \
	fi
	@ PROTOC_VERSION=$(shell protoc --version | grep ^libprotoc | sed 's/^.* //g'); \
	if ! echo $PROTOC_VERSION | grep -rq "3.0.0"; then \
		echo "protoc version must be 3.0.0" >&2; \
		exit 1; \
	fi

	#go get -u github.com/golang/protobuf/protoc-gen-go;
	#go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway;
	#./scripts/build/buildProto.sh

	@ if [ $$? -ne 0 ]; then \
		echo "failed to build protobuf files" >&2; \
		exit 1; \
	fi

vpservice: protos
	go build -a -o $(BIN_DIR)/vpservice ./cmd/vpservice;

vpservice-docker: vpservice
	./scripts/build/buildVPService.sh

revtr: protos
	go build -a -o $(BIN_DIR)/revtr ./cmd/revtr;

revtr-docker: revtr
	./scripts/build/buildRevtr.sh

atlas: protos
	go build -a -o $(BIN_DIR)/atlas ./cmd/atlas;

atlas-docker: atlas
	./scripts/build/buildAtlas.sh

plcontroller: protos
	go build -a -o $(BIN_DIR)/plcontroller ./cmd/plcontroller;

plcontroller-docker: plcontroller
	./scripts/build/buildPlc.sh

controller: protos
	go build -a -o $(BIN_DIR)/controller ./cmd/controller;

controller-docker: controller
	./scripts/build/buildCC.sh

plvp: protos
	go build -a -o $(BIN_DIR)/plvp ./cmd/plvp;

plvp-docker: plvp
	./scripts/build/buildPLVP.sh


build-tools: deps protos
	@ for dir in $(TOOL_DIRS); do \
		echo "go build -a -o $(BIN_DIR)/`basename $$dir` ./$$dir"; \
		go build -a -o $(BIN_DIR)/`basename $$dir` ./$$dir; \
	done

clean:
	@ for dir in $(MAIN_DIRS); do \
		echo "go clean -i -r ./$$dir"; \
		go clean -i -r ./$$dir; \
	done
	go clean -i -r ./cmd/plcontroller;
	go clean -i -r ./cmd/controller;
	@ for dir in $(MAIN_DIRS); do \
		echo "rm $(BIN_DIR)/`basename $$dir`"; \
		rm -f $(BIN_DIR)/`basename $$dir`; \
	done

test-all:
	@ for dir in $(TEST_DIRS); do \
		go test -v ./$$dir; \
	done

.PHONY: \
	deps \
	protos \
	vpservice \
	revtr \
	atlas \
	plcontroller \
	controller \
	plvp \
	all \
	clean \
	test-all \
	build-tools 
