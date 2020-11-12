SHELL=/bin/bash -o pipefail
APP=secrets-manager
PACKAGE=secrets-manager
BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_REMOTE             = origin
GIT_BRANCH             = $(shell git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')
GIT_TAG                = $(shell git describe --exact-match --tags HEAD 2>/dev/null || git rev-parse --short=8 HEAD 2>/dev/null)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

export DOCKER_BUILDKIT = 1

# To allow you to build with or without cache for debugging purposes.
DOCKER_BUILD_OPTS     := --no-cache
# Use a different Dockerfile, e.g. for building for Windows or dev images.
DOCKERFILE            := deploy/Dockerfile
DOCKER_REPOSITORY	  := lethe3000

# The rules for what version are, in order of precedence
# 1. If anything passed at the command line (e.g. make release VERSION=...)
# 2. If on master, it must be "latest".
# 3. If on tag, must be tag.
# 4. If on a release branch, the most recent tag that contain the major minor on that branch,
# 5. Otherwise, the branch.
#
VERSION := $(subst /,-,$(GIT_BRANCH))

ifeq ($(GIT_BRANCH),master)
VERSION := latest
endif

ifneq ($(findstring release,$(GIT_BRANCH)),)
VERSION := $(shell git tag --points-at=HEAD|grep ^v|head -n1)
endif

override LDFLAGS += \
  -X ${PACKAGE}/version.version=${VERSION} \
  -X ${PACKAGE}/version.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}/version.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}/version.gitTreeState=${GIT_TREE_STATE}

.PHONY: build docker_image
build:
	CGO_ENABLED=0 go build -ldflags '${LDFLAGS}' -o ${APP} cmd/secrets-manager/main.go
	./${APP} version

docker_image:
	docker build -t ${DOCKER_REPOSITORY}/${APP}:${VERSION} --build-arg VERSION=${VERSION} --build-arg BUILD_DATE=${BUILD_DATE} --build-arg GIT_COMMIT=${GIT_COMMIT} --build-arg GIT_TREE_STATE=${GIT_TREE_STATE} -f deploy/Dockerfile .
	docker push ${DOCKER_REPOSITORY}/${APP}:${VERSION}
