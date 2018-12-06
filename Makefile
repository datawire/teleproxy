all: build check

# Since we're using Go modules, we don't need to set GOPATH to build.
# But we would like to set it be somewhere under the current
# directory, so that the module cache (${GOPATH}/pkg/mod) gets copied
# in to the Docker context.  Speeding up Docker is the only reason we
# care about setting GOPATH or GOCACHE.
export GOPATH = $(CURDIR)/.gocache/workspace
export GOCACHE = $(CURDIR)/.gocache/go-build

include build-aux/common.mk
include build-aux/go-mod.mk
include build-aux/kubernaut.mk

export PATH:=$(CURDIR)/bin_$(GOOS)_$(GOARCH):$(PATH)
export KUBECONFIG=${PWD}/cluster.knaut

manifests: cluster.knaut bin_$(GOOS)_$(GOARCH)/kubeapply
	bin_$(GOOS)_$(GOARCH)/kubeapply -f k8s
.PHONY: manifests

claim: cluster.knaut.clean cluster.knaut

shell: cluster.knaut
	@exec env -u MAKELEVEL PS1="(dev) [\W]$$ " bash

other-tests: build
	go test -v $$(go list ./... | grep -vF -e $(go.module)/internal/pkg/nat -e $(go.module)/cmd/teleproxy)

nat-tests: build
	go test -v -exec sudo $(go.module)/internal/pkg/nat

smoke-tests: build manifests
	go test -v -exec "sudo env PATH=${PATH} KUBECONFIG=${KUBECONFIG}" $(go.module)/cmd/teleproxy

sudo-tests: nat-tests smoke-tests

run-tests: sudo-tests other-tests

test-go: go-get run-tests

test-docker: build
ifneq ($(shell which docker 2>/dev/null),)
test-docker: $(addprefix bin_linux_amd64/,$(notdir $(go.bins)))
	docker build -f scripts/Dockerfile . -t teleproxy-make
	docker run --cap-add=NET_ADMIN teleproxy-make nat-tests
else
	@echo "SKIPPING DOCKER TESTS"
endif

test: test-go test-docker
check: test

run: build
	bin_$(GOOS)_$(GOARCH)/teleproxy

clean: cluster.knaut.clean

clobber:
	find .gocache/workspace -exec chmod +w {} + || true
	rm -rf .gocache
