.SECONDEXPANSION:

ifeq ($(OS),Windows_NT)
  SHELL := powershell.exe
  .SHELLFLAGS := -NoProfile -Command
  MKDIR := $$null = New-Item -Type Directory
  COPY := Copy-Item -Force
  MOVE := Move-Item -Force
  RM := Remove-Item -Recurse -Force
else
  MKDIR := mkdir -p
  COPY := cp -f
  MOVE := mv -f
  RM := rm -rf
endif

CMD_PATH := cmd/git-remote-https+iap
CMD_NAME := git-remote-https+iap
BUILD_TARGETS := \
	darwin-amd64 \
   	darwin-arm64 \
	linux-amd64 \
	windows-amd64

DIST_PATH := dist/
BIN_PATH := $(DIST_PATH)bin/
RELEASE_PATH := $(DIST_PATH)releases/

version := $(shell git describe --match "v*.*" --abbrev=7 --tags --dirty)
build_args := -ldflags "-X main.version=${version}"

.PHONY: all
all: build

$(RELEASE_PATH) $(addprefix $(BIN_PATH), $(BUILD_TARGETS)):
	$(MKDIR) $@

$(BIN_PATH)%/$(CMD_NAME): export GOOS   = $(word 1, $(subst -, ,$*))
$(BIN_PATH)%/$(CMD_NAME): export GOARCH = $(word 2, $(subst -, ,$*))
$(BIN_PATH)%/$(CMD_NAME): export CGO_ENABLED = 0
$(BIN_PATH)%/$(CMD_NAME): $(wildcard $(CMD_PATH)/*.go internal/*/*.go) | $(BIN_PATH)%
	go build $(build_args) -o $@ $<

$(BIN_PATH)%/$(CMD_NAME).exe: $(BIN_PATH)%/$(CMD_NAME)
	$(MOVE) $< $@

BUILDS := $(foreach target, $(BUILD_TARGETS), $(BIN_PATH)$(target)/$(CMD_NAME)$(if $(filter windows%,$(target)),.exe))

.PHONY: build
build: $(BUILDS)

RELEASE_INCLUDES = README.md
RELEASE_TARGETS := $(foreach target, $(BUILD_TARGETS), $(RELEASE_PATH)$(CMD_NAME)-$(target)-$(version).tar.gz)

$(RELEASE_PATH)$(CMD_NAME)-%-$(version).tar.gz: $(BIN_PATH)%/$(CMD_NAME).tar.gz | $(RELEASE_PATH)
	$(MOVE) $< $@

$(BIN_PATH)%/$(CMD_NAME).tar.gz: $(BIN_PATH)%/$(CMD_NAME)$$(if $$(filter windows%,$$*),.exe) $(RELEASE_INCLUDES)
	$(COPY) $(filter-out $<,$^) $(@D)
	cd $(@D); tar czf $(@F) $(^F)

%.sha256: %
ifeq ($(OS),Windows_NT)
	$$env:PSModulePath = "$$PSHOME\\Modules"; "$$((Get-FileHash -Algorithm SHA256 $<).Hash.ToLower())  $(<F)" > $@
else
	cd $(@D) && shasum -a 256 $(<F) > $(@F)
endif

.PHONY: release
release: $(RELEASE_TARGETS) $(addsuffix .sha256, $(RELEASE_TARGETS))

.PHONY: version
version:
	@echo "$(version)"

.PHONY: clean
clean:
	-$(RM) $(DIST_PATH)
