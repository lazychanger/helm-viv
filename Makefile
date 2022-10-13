PLUGIN_NAME := viv

VERSION?=$(shell yq ".version" "plugin.yaml")

CURRENT_DIR?=$(shell pwd)

PACKAGE=$(shell cat ${CURRENT_DIR}/go.mod | head -n 1 | sed  "s/module //g")

#VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)
VOLUME_MOUNT=$(shell if test "$(go env GOOS)" = "darwin"; then echo ":delegated"; elif test selinuxenabled; then echo ":delegated"; else echo ""; fi)

COMMAND=

STATIC_BUILD?=true

LDFLAGS = \
  -X ${PACKAGE}/common.version=${VERSION} \
  -X ${PACKAGE}/common.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}/common.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}/common.gitTreeState=${GIT_TREE_STATE} \


ifeq (${STATIC_BUILD}, true)
override LDFLAGS += -extldflags "-static"
endif

ifneq (${GIT_TAG},)
IMAGE_TAG=${GIT_TAG}
LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
else
IMAGE_TAG?=latest
endif

.PHONY: build
build: build_linux build_mac build_windows

.PHONY: link
link: link_linux link_mac link_windows

.PHONY: build_windows
build_windows: export GOARCH=amd64
build_windows: export GO111MODULE=on
build_windows:
	@GOOS=windows go build -v --ldflags="-w ${LDFLAGS}" \
		-o bin/windows/amd64/helm-viv cmd/helm-variable-in-values/main.go   # windows

.PHONY: link_windows
link_windows:
	@cp bin/windows/amd64/helm-viv ./bin/helm-viv

.PHONY: build_linux
build_linux: export GOARCH=amd64
build_linux: export CGO_ENABLED=0
build_linux: export GO111MODULE=on
build_linux:
	@GOOS=linux go build -v --ldflags="-w ${LDFLAGS}" \
		-o bin/linux/amd64/helm-viv cmd/helm-variable-in-values/main.go   # window  # linux

.PHONY: link_linux
link_linux:
	@cp bin/linux/amd64/helm-viv ./bin/helm-cm-push

.PHONY: build_mac
build_mac: export GOARCH=amd64
build_mac: export CGO_ENABLED=0
build_mac: export GO111MODULE=on
build_mac:
	@GOOS=darwin go build -v --ldflags="-w ${LDFLAGS}" \
		-o bin/darwin/amd64/helm-viv cmd/helm-variable-in-values/main.go   # window # mac osx
	@cp bin/darwin/amd64/helm-viv ./bin/helm-viv # For use w make install

.PHONY: link_mac
link_mac:
	@cp bin/darwin/amd64/helm-viv ./bin/helm-viv

.PHONY: clean
clean:
	@git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: covhtml
covhtml:
	@go tool cover -html=.cover/cover.out

.PHONY: tree
tree:
	@tree -I vendor

.PHONY: release
release:
	@scripts/release.sh $(VERSION)

.PHONY: install
install:
	HELM_PUSH_PLUGIN_NO_INSTALL_HOOK=1 helm plugin install $(shell pwd)

.PHONY: remove
remove:
	helm plugin remove $(PLUGIN_NAME)

.PHONY: setup-test-environment