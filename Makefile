PLUGIN_NAME := viv

VERSION?=$(shell yq ".version" "plugin.yaml")

.PHONY: build
build: build_linux build_mac build_windows

build_windows: export GOARCH=amd64
build_windows: export GO111MODULE=on
build_windows:
	@GOOS=windows go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/windows/amd64/helm-viv cmd/helm-variable-in-values/main.go  # windows

link_windows:
	@cp bin/windows/amd64/helm-viv ./bin/helm-viv

build_linux: export GOARCH=amd64
build_linux: export CGO_ENABLED=0
build_linux: export GO111MODULE=on
build_linux:
	@GOOS=linux go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/linux/amd64/helm-viv cmd/helm-variable-in-values/main.go  # linux

link_linux:
	@cp bin/linux/amd64/helm-viv ./bin/helm-cm-push

build_mac: export GOARCH=amd64
build_mac: export CGO_ENABLED=0
build_mac: export GO111MODULE=on
build_mac:
	@GOOS=darwin go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/darwin/amd64/helm-viv cmd/helm-variable-in-values/main.go # mac osx
	@cp bin/darwin/amd64/helm-viv ./bin/helm-viv # For use w make install

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