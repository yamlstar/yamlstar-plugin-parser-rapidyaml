MAKES-COMMIT := a7b80ec8f10ac700693278a559f315fccd50b5ad
M ?= .cache/makes
$(shell test -d $M || { \
  git clone -q https://github.com/makeplus/makes $M && \
  git -C $M checkout -q $(MAKES-COMMIT); \
})

GO-VERSION := 1.27.1
PERL-VERSION := 5.44.0.0
export UV_CACHE_DIR := $(CURDIR)/.cache/uv

include $M/init.mk
include $M/gh.mk
include $M/go.mk
include $M/perl.mk
include $M/docker.mk
include $M/clean.mk
include $M/shellcheck.mk

# Correct uv-managed CPython names until Makes provides these mappings.
CPYTHON-OA-linux-arm64 := linux-aarch64-gnu
CPYTHON-OA-macos-int64 := macos-x86_64-none
ifneq ($(CPYTHON-OA-$(OS-ARCH)),)
override PYTHON-NAME = \
  cpython-$(PYTHON-VERSION)-$(CPYTHON-OA-$(OS-ARCH))
endif
include $M/python.mk
include $M/shell.mk

PYTHON-VENV-SETUP := \
  $(UV) pip install --python $(PYTHON-VENV) setuptools wheel

VERSION := 0.1.4
MODULE := github.com/yamlstar/yamlstar-plugin-parser-rapidyaml
PLUGIN := yamlstar-plugin-parser-rapidyaml
LIB-NAME := lib$(PLUGIN).$(SO)
LIB := lib/$(LIB-NAME)
RELEASE_LIB_DIR := .cache/release/lib
RELEASE_LIB := $(RELEASE_LIB_DIR)/$(LIB-NAME)
NATIVE-SOURCES := plugin.cpp rymlshim.cpp rymlshim.h include/yamlstar_plugin.h
NATIVE-MANIFEST := .cache/native-manifest.h
NATIVE-CXXFLAGS := -std=c++11 -DNDEBUG -fPIC -fvisibility=hidden \
  -DC4_EXCEPTIONS_ENABLED -DC4_ERROR_THROWS_EXCEPTION \
  -I.cache/rapidyaml/src -I.cache/rapidyaml/src_extra \
  -I.cache/rapidyaml/ext/c4core.src
ifeq ($(IS-MACOS),true)
# Match the existing wheel deployment targets for each macOS architecture.
NATIVE-CXXFLAGS += -mmacosx-version-min=$(if $(IS-INTEL),15.0,14.0)
NATIVE-LDFLAGS := -dynamiclib -Wl,-install_name,@rpath/$(LIB-NAME)
else
NATIVE-LDFLAGS := -shared -pthread
endif
WHEEL-PACKAGE := yamlstar_plugin_parser_rapidyaml
WHEEL-PACKAGE-DIR := python/lib/$(WHEEL-PACKAGE)
WHEEL-LIB-DIR := $(WHEEL-PACKAGE-DIR)/lib
WHEEL-CACHE := .cache/wheel
WHEEL-SOURCE-LIB = \
  $(WHEEL-CACHE)/$(RELEASE_NAME)/lib/$(LIB-NAME)
WHEEL-PLAT-linux-x64 := manylinux_2_28_x86_64
WHEEL-PLAT-linux-aarch64 := manylinux_2_28_aarch64
WHEEL-PLAT-macos-x64 := macosx_15_0_x86_64
WHEEL-PLAT-macos-arm64 := macosx_14_0_arm64
WHEEL-PLAT = $(WHEEL-PLAT-$(RELEASE_PLATFORM))
WHEEL-TAG = py3-none-$(WHEEL-PLAT)
WHEEL-FILE = python/dist/$(WHEEL-PACKAGE)-$(VERSION)-$(WHEEL-TAG).whl
WHEEL ?= $(WHEEL-FILE)
WHEEL-TEST-VENV := .cache/wheel-test
WHEEL-TEST-PYTHON := $(WHEEL-TEST-VENV)/bin/python

RELEASE-ARCH := $(strip $(if $(IS-INTEL),x64,\
  $(if $(IS-LINUX),aarch64,arm64)))
RELEASE_PLATFORM ?= $(OS-NAME)-$(RELEASE-ARCH)
RELEASE_NAME := $(PLUGIN)-$(VERSION)-$(RELEASE_PLATFORM)
WHEEL-STAGE := $(WHEEL-CACHE)/$(RELEASE_NAME)/.staged
RELEASE_DIR := dist/$(RELEASE_NAME)
ARCHIVE := dist/$(RELEASE_NAME).tar.xz
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct 2>/dev/null || echo 0)
TAR ?= $(if $(IS-MACOS),gtar,tar)
MANYLINUX-REPO-linux-x64 := quay.io/pypa/manylinux_2_28_x86_64
MANYLINUX-REPO-linux-aarch64 := quay.io/pypa/manylinux_2_28_aarch64
MANYLINUX-DIGEST-linux-x64 := \
  sha256:53390351aeb4688114b02c36a23b3e6ce1166ee9b7afc5df1a4f776354fc764c
MANYLINUX-DIGEST-linux-aarch64 := \
  sha256:ad74e53b713f3b07d8c889c526dc0c6500da9827b45e38739570875fef52e28f
MANYLINUX-REPO := $(MANYLINUX-REPO-$(RELEASE_PLATFORM))
MANYLINUX-DIGEST := $(MANYLINUX-DIGEST-$(RELEASE_PLATFORM))
MANYLINUX-IMAGE := $(MANYLINUX-REPO)@$(MANYLINUX-DIGEST)
RELEASE-REPO := yamlstar/yamlstar-plugin-parser-rapidyaml
RELEASE-WORKFLOW := release.yaml
RELEASE-SCRIPT := util/release
RELEASE-CMD = \
  PERL=$(PERL-LOCAL)/bin/perl \
  GH=$(GH) \
  RELEASE_REPO=$(RELEASE-REPO) \
  RELEASE_WORKFLOW=$(RELEASE-WORKFLOW) \
  $(RELEASE-SCRIPT)

MAKES-CLEAN := \
  $(WHEEL-CACHE) \
  $(WHEEL-TEST-VENV) \
  $(RELEASE_LIB_DIR) \
  dist \
  lib \
  python/build \
  python/dist \
  python/*.egg-info \
  python/lib/*.egg-info \
  $(WHEEL-LIB-DIR) \
  $(WHEEL-PACKAGE-DIR)/plugin.edn \
  $(WHEEL-PACKAGE-DIR)/License \
  $(WHEEL-PACKAGE-DIR)/licenses

default:: build

build: $(LIB)

test: $(LIB) $(GO) $(SHELLCHECK) $(PERL)
	$(GO) test ./...
	$(call compile-abi-test,.cache/abi-test)
	YAMLSTAR_LIBRARY_PATH=$(abspath lib) .cache/abi-test
	$(SHELLCHECK) util/release util/test-release util/test-goyaml-cli \
	  util/test-archive util/fetch-rapidyaml
	PERL=$(PERL-LOCAL)/bin/perl util/test-release

test-goyaml: $(LIB) $(GO)
	CGO_ENABLED=0 $(GO) test ./goyaml

test-goyaml-race: $(LIB) $(GO)
	CGO_ENABLED=1 $(GO) test -race ./goyaml

test-goyaml-cli: $(LIB) $(GO) $(PERL) $(PYTHON)
	GO=$(GO) PERL=$(PERL-LOCAL)/bin/perl PYTHON=$(PYTHON) \
	  util/test-goyaml-cli

benchmark: $(LIB) $(GO)
	$(GO) test -run '^$$' -bench . -count=7

wheel: dist
	$(MAKE) -o $(ARCHIVE) $(WHEEL-FILE) VERSION=$(VERSION) \
	  RELEASE_PLATFORM=$(RELEASE_PLATFORM)

test-wheel: $(UV) $(PYTHON)
	test -f "$(WHEEL)"
	@test -n "$(WHEEL-PLAT)" || { \
	  echo "No wheel platform for $(RELEASE_PLATFORM)" >&2; \
	  exit 1; \
	}
	rm -rf $(WHEEL-CACHE)
	mkdir -p $(WHEEL-CACHE)
	tar -C $(WHEEL-CACHE) -xf $(ARCHIVE)
	test -f $(WHEEL-SOURCE-LIB)
	rm -rf $(WHEEL-TEST-VENV)
	$(UV) venv --python $(PYTHON) $(WHEEL-TEST-VENV)
	$(UV) pip install --python $(WHEEL-TEST-PYTHON) \
	  --force-reinstall $(WHEEL)
	$(call compile-abi-test,.cache/wheel-abi-test)
	library_dir="$$( \
	  $(WHEEL-TEST-PYTHON) python/test_wheel.py \
	    $(WHEEL) $(WHEEL-TAG) $(WHEEL-SOURCE-LIB) \
	    $(VERSION) \
	)"; \
	YAMLSTAR_LIBRARY_PATH="$$library_dir" .cache/wheel-abi-test

test-release: $(RELEASE_LIB)
	$(call compile-abi-test,.cache/release-abi-test)
	YAMLSTAR_LIBRARY_PATH=$(abspath $(RELEASE_LIB_DIR)) \
	  .cache/release-abi-test

test-archive: $(PERL)
	test -n "$(ARCHIVE)"
	PERL=$(PERL-LOCAL)/bin/perl CC="$(CC)" util/test-archive \
	  "$(ARCHIVE)" "$(RELEASE_PLATFORM)" "$(VERSION)"

release-check: .cache/rapidyaml/.pinned
	[[ "$(VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]]
	grep -Fq ':version "$(VERSION)"' plugin.edn
	test "$$(git -C $M rev-parse HEAD)" = "$(MAKES-COMMIT)"
	case "$(RELEASE_PLATFORM)" in \
	  linux-x64|linux-aarch64|macos-x64|macos-arm64) ;; \
	  *) echo "Unsupported release platform: $(RELEASE_PLATFORM)" >&2; \
	     exit 1 ;; \
	esac

ifeq ($(IS-LINUX),true)
dist: test release-linux
else
dist: test release-archive
endif

release-linux: | $(DOCKER)
	test -n "$(MANYLINUX-IMAGE)"
	$(DOCKER) run --rm \
	  -e VERSION="$(VERSION)" \
	  -e RELEASE_PLATFORM="$(RELEASE_PLATFORM)" \
	  -e BUILD_UID="$$(id -u)" \
	  -e BUILD_GID="$$(id -g)" \
	  -v "$(CURDIR):/work" \
	  -w /work \
	  "$(MANYLINUX-IMAGE)" \
	  bash -c ' \
	    set -euo pipefail; \
	    cleanup() { \
	      for path in /work/.cache /work/dist; do \
	        test ! -e "$$path" || \
	          chown -R "$$BUILD_UID:$$BUILD_GID" "$$path"; \
	      done; \
	    }; \
	    trap cleanup EXIT; \
	    dnf install -y binutils curl file gcc gcc-c++ git make perl unzip xz >/dev/null; \
	    git config --global --add safe.directory /work; \
	    git config --global --add safe.directory /work/.cache/makes; \
	    git config --global --add safe.directory /work/.cache/rapidyaml; \
	    make release-archive \
	      VERSION="$$VERSION" \
	      RELEASE_PLATFORM="$$RELEASE_PLATFORM" \
	  '

release-archive: $(ARCHIVE)

export OLD_VERSION := $o
export NEW_VERSION := $(or $v,$n)
ifdef d
export YS_RELEASE_DRYRUN := 1
endif
ifdef a
export YS_RELEASE_ALLOW_BRANCH := 1
endif

release: $(PERL) $(GH)
ifndef v
	$(error 'make release' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) release "$(o)" "$(v)"

release-list: $(PERL)
	$(RELEASE-CMD) list

release-sanity-check: $(PERL)
ifndef v
	$(error 'make release-sanity-check' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) sanity-check "$(o)" "$(v)"

release-version-bump: $(PERL)
ifndef v
	$(error 'make release-version-bump' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) version-bump "$(o)" "$(v)"

release-pull: $(PERL)
	$(RELEASE-CMD) pull

release-commit: $(PERL)
ifndef v
	$(error 'make release-commit' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) commit "$(v)"

release-tag: $(PERL)
ifndef v
	$(error 'make release-tag' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) tag "$(v)"

release-push: $(PERL)
ifndef v
	$(error 'make release-push' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) push "$(v)"

release-build-github: $(PERL) $(GH)
ifndef v
	$(error 'make release-build-github' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) build-github "$(v)"

release-retry: $(PERL) $(GH)
ifndef v
	$(error 'make release-retry' requires v=NEW_VERSION)
endif
	$(RELEASE-CMD) retry "$(v)"

$(ARCHIVE): Makefile release-check test-release plugin.edn \
  include/yamlstar_plugin.h License ReadMe.md Performance.md util/test-archive \
  $(wildcard licenses/*)
	rm -rf "$(RELEASE_DIR)" "$@"
	install -d \
	  "$(RELEASE_DIR)/lib" \
	  "$(RELEASE_DIR)/include/yamlstar" \
	  "$(RELEASE_DIR)/share/yamlstar/plugins/rapidyaml"
	install -m 755 "$(RELEASE_LIB)" \
	  "$(RELEASE_DIR)/lib/$(LIB-NAME)"
	install -m 644 include/yamlstar_plugin.h \
	  "$(RELEASE_DIR)/include/yamlstar/"
	install -m 644 plugin.edn \
	  "$(RELEASE_DIR)/share/yamlstar/plugins/rapidyaml/"
	install -m 644 License ReadMe.md Performance.md "$(RELEASE_DIR)/"
	cp -R licenses "$(RELEASE_DIR)/"
	COPYFILE_DISABLE=1 $(TAR) \
	  --sort=name \
	  --mtime="@$(SOURCE_DATE_EPOCH)" \
	  --owner=0 --group=0 --numeric-owner \
	  -C dist -cJf "$@" "$(RELEASE_NAME)"
	$(MAKE) test-archive ARCHIVE="$@" \
	  RELEASE_PLATFORM="$(RELEASE_PLATFORM)" VERSION="$(VERSION)"

$(WHEEL-STAGE): $(ARCHIVE) plugin.edn License python/setup.py \
  $(wildcard licenses/*) \
  python/lib/$(WHEEL-PACKAGE)/__init__.py
	@test -n "$(WHEEL-PLAT)" || { \
	  echo "No wheel platform for $(RELEASE_PLATFORM)" >&2; \
	  exit 1; \
	}
	rm -rf $(WHEEL-CACHE) $(WHEEL-LIB-DIR)
	mkdir -p $(WHEEL-CACHE) $(WHEEL-LIB-DIR)
	tar -C $(WHEEL-CACHE) -xf $(ARCHIVE)
	cp -p $(WHEEL-CACHE)/$(RELEASE_NAME)/lib/$(LIB-NAME) \
	  $(WHEEL-LIB-DIR)/
	cp -p plugin.edn $(WHEEL-PACKAGE-DIR)/
	cp -p License $(WHEEL-PACKAGE-DIR)/
	cp -R licenses $(WHEEL-PACKAGE-DIR)/
	touch $@

$(WHEEL-FILE): $(WHEEL-STAGE) $(PYTHON-VENV)
	rm -rf python/build python/dist python/*.egg-info \
	  python/lib/*.egg-info
	cd python && $(VENV) && \
	  python setup.py bdist_wheel --plat-name $(WHEEL-PLAT)
	test -f $@

RAPIDYAML-COMMIT := f8ac8dd50f4f7916579d55a05ebf9c6488e52670

.cache/rapidyaml/.pinned: util/fetch-rapidyaml Makefile
	bash util/fetch-rapidyaml $(RAPIDYAML-COMMIT)

# Use Perl directly because the host's bin/perl symlink is invalid in Docker.
$(NATIVE-MANIFEST): plugin.edn Makefile $(PERL)
	@mkdir -p $(dir $@)
	$(PERL-LOCAL)/bin/perl -0777 -ne \
	  'print qq{static const char plugin_manifest[] = R"YS(}, $$_, qq{)YS";\n}' \
	  $< > $@

$(LIB): .cache/rapidyaml/.pinned $(NATIVE-SOURCES) $(NATIVE-MANIFEST)
	@mkdir -p $(dir $@)
	$(CXX) $(CPPFLAGS) $(CXXFLAGS) $(NATIVE-CXXFLAGS) -O2 \
	  $(NATIVE-LDFLAGS) -o $@ plugin.cpp rymlshim.cpp $(LDFLAGS)

$(RELEASE_LIB): .cache/rapidyaml/.pinned $(NATIVE-SOURCES) $(NATIVE-MANIFEST)
	@mkdir -p $(dir $@)
	$(CXX) $(CPPFLAGS) $(CXXFLAGS) $(NATIVE-CXXFLAGS) -O2 \
	  $(NATIVE-LDFLAGS) -o $@ plugin.cpp rymlshim.cpp $(LDFLAGS)

define compile-abi-test
	$(CC) -Wall -Wextra -Werror -Iinclude \
	  -DPLUGIN_EXTENSION='"$(SO)"' \
	  -DPLUGIN_VERSION='"$(VERSION)"' \
	  test/abi.c $(if $(IS-MACOS),,-ldl) -pthread -o $(1)
endef

format: $(GO)
	$(GO) fmt ./...

install: $(LIB)
	install -d $(PREFIX)/lib
	install -m 755 $(LIB) $(PREFIX)/lib/

YAML-TEST-SUITE-COMMIT := 6ad3d2c62885d82fc349026c136ef560838fdf3d

.cache/yaml-test-suite/.pinned:
	if [[ ! -d .cache/yaml-test-suite/.git ]]; then \
	  git clone --no-checkout https://github.com/yaml/yaml-test-suite \
	    .cache/yaml-test-suite; \
	fi
	git -C .cache/yaml-test-suite checkout $(YAML-TEST-SUITE-COMMIT)
	touch $@

test-conformance: $(LIB) $(GO) .cache/yaml-test-suite/.pinned
	YAMLSTAR_TEST_SUITE_DIR=.cache/yaml-test-suite \
	  $(GO) test -run TestConformance -v

fuzz: $(LIB) $(GO)
	$(GO) test -run '^$$' -fuzz FuzzParse -fuzztime=10s -parallel=2
