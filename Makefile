.PHONY: all build test

# This Makefile is a simple example that demonstrates usual steps to build a binary that can be run in the same
# architecture that was compiled in. The "ldflags" in the build assure that any needed dependency is included in the
# binary and no external dependencies are needed to run the service.

BIN_NAME :=sonic
OS := $(shell uname | tr '[:upper:]' '[:lower:]')
VERSION := 1.0.0
PKG_NAME := sonic
LICENSE := Apache License, Version 2.0
VENDOR=
URL := http://sonic.io
RELEASE := 0
USER := sonic
ARCH := amd64
DESC := High performance API gateway.
MAINTAINER := Huy Duc Dao <ducdh.starvn@gmail.com>
DOCKER_WORK_DIR := /tmp/fpm
DOCKER_FPM := starvn/fpm
GOLANG_VERSION := 1.17

FPM_OPTS=-s dir -v $(VERSION) -n $(PKG_NAME) \
  --license "$(LICENSE)" \
  --vendor "$(VENDOR)" \
  --maintainer "$(MAINTAINER)" \
  --architecture $(ARCH) \
  --url "$(URL)" \
  --description  "$(DESC)" \
	--config-files etc/ \
  --verbose

DEB_OPTS= -t deb --deb-user $(USER) \
	--depends ca-certificates \
	--before-remove builder/script/prerm.deb \
  --after-remove builder/script/postrm.deb \
	--before-install builder/script/preinst.deb

RPM_OPTS =--rpm-user $(USER) \
	--before-install builder/script/preinst.rpm \
	--before-remove builder/script/prerm.rpm \
  --after-remove builder/script/postrm.rpm

DEBNAME=${PKG_NAME}_${VERSION}-${RELEASE}_${ARCH}.deb
RPMNAME=${PKG_NAME}-${VERSION}-${RELEASE}.x86_64.rpm

all: test

update_sonic_deps:
	go get github.com/starvn/turbo@v1.0.3
	go get github.com/starvn/go-bloom-filter@v1.0.0
	make test

build:
	@echo "Building the binary..."
	@go get .
	@go build -ldflags="-X github.com/starvn/turbo/core.SonicVersion=${VERSION}" -o ${BIN_NAME} ./cmd/gateway
	@echo "You can now use ./${BIN_NAME}"

test: build
	go test -v ./test

build_on_docker:
	docker run --rm -it -v "${PWD}:/app" -w /app golang:${GOLANG_VERSION} make build

docker:
	docker build --pull -t starvn/sonic:${VERSION} .

builder/skel/%/etc/init.d/sonic: builder/file/sonic.init
	mkdir -p "$(dir $@)"
	cp builder/file/sonic.init "$@"

builder/skel/%/usr/bin/sonic: sonic
	mkdir -p "$(dir $@)"
	cp sonic "$@"

builder/skel/%/etc/sonic/sonic.json: sonic.json
	mkdir -p "$(dir $@)"
	cp sonic.json "$@"

builder/skel/%/lib/systemd/system/sonic.service: builder/file/sonic.service
	mkdir -p "$(dir $@)"
	cp builder/file/sonic.service "$@"

builder/skel/%/usr/lib/systemd/system/sonic.service: builder/file/sonic.service
	mkdir -p "$(dir $@)"
	cp builder/file/sonic.service "$@"

.PHONE: tgz
tgz: builder/skel/tgz/usr/bin/sonic
tgz: builder/skel/tgz/etc/sonic/sonic.json
tgz: builder/skel/tgz/etc/init.d/sonic
	tar zcvf sonic_${VERSION}_${ARCH}.tar.gz -C builder/skel/tgz/ .

.PHONY: deb
deb: builder/skel/deb/usr/bin/sonic
deb: builder/skel/deb/etc/sonic/sonic.json
	docker run --rm -it -v "${PWD}:${DOCKER_WORK_DIR}" -w ${DOCKER_WORK_DIR} ${DOCKER_FPM}:deb -t deb ${DEB_OPTS} \
		--iteration ${RELEASE} \
		--deb-systemd builder/file/sonic.service \
		-C builder/skel/deb \
		${FPM_OPTS}

.PHONY: rpm
rpm: builder/skel/rpm/usr/lib/systemd/system/sonic.service
rpm: builder/skel/rpm/usr/bin/sonic
rpm: builder/skel/rpm/etc/sonic/sonic.json
	docker run --rm -it -v "${PWD}:${DOCKER_WORK_DIR}" -w ${DOCKER_WORK_DIR} ${DOCKER_FPM}:rpm -t rpm ${RPM_OPTS} \
		--iteration ${RELEASE} \
		-C builder/skel/rpm \
		${FPM_OPTS}


.PHONY: clean
clean:
	rm -rf builder/skel/*
	rm -f *.deb
	rm -f *.rpm
	rm -f *.tar.gz
	rm -f sonic
	rm -rf vendor/
