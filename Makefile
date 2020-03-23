CMD_PATH := cmd/git-remote-https+iap
CMD_NAME := git-remote-https+iap
BUILD_TARGETS := \
	darwin-amd64 \
	linux-amd64

DIST_PATH := dist/
BIN_PATH := $(DIST_PATH)bin/
RELEASE_PATH := $(DIST_PATH)releases/

version := $(shell git describe --match "v*.*" --abbrev=7 --tags --dirty)
build_args := -ldflags "-X main.version=${version}"
tar_xform_arg := $(shell tar --version | grep -q 'GNU tar' && echo '--xform' || echo '-s')
tar_xform_cmd := $(shell tar --version | grep -q 'GNU tar' && echo 's')

.PHONY: all
all: build

$(BIN_PATH) $(RELEASE_PATH):
	mkdir -p $@

BUILDS := $(foreach target, $(BUILD_TARGETS), $(BIN_PATH)$(CMD_NAME)-$(target))
$(BUILDS): OS   = $(word 1, $(subst -, ,$(subst $(CMD_NAME)-,,$(notdir $@))))
$(BUILDS): ARCH = $(word 2, $(subst -, ,$(subst $(CMD_NAME)-,,$(notdir $@))))
$(BUILDS): $(BIN_PATH)
	env CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build ${build_args} -o $(BIN_PATH)${CMD_NAME}-$(OS)-$(ARCH) ${CMD_PATH}/*.go
build: $(BUILDS)

RELEASE_INCLUDES = README.md
RELEASE_TARGETS := $(foreach target, $(BUILDS), $(RELEASE_PATH)$(notdir $(target))-$(version).tar.gz)
$(RELEASE_TARGETS): $(RELEASE_PATH)%-$(version).tar.gz: $(BIN_PATH)% $(RELEASE_INCLUDES)
	mkdir -p $(RELEASE_PATH)
	tar $(tar_xform_arg) '$(tar_xform_cmd)!$(BIN_PATH)$(CMD_NAME).*!$(CMD_NAME)!' -czf $@ $^
	cd $(RELEASE_PATH) && shasum -a 256 $(notdir $@) >$(notdir $@).sha256
release: $(RELEASE_TARGETS)

.PHONY: version
version:
	@echo "$(version)"

.PHONY: clean
clean:
	rm -rf $(DIST_PATH)
