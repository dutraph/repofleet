# Makefile for fleet — generated from the tui-cli-pattern skill.
# Customize PREFIX or VERSION via command line:
#   make install                   → auto-bumps the patch
#   make install VERSION=1.0.0     → pins to that version
#   make install PREFIX=$HOME/.local

BIN := fleet
PKG := ./cmd/fleet

PREFIX  ?= /usr/local
BINDIR  := $(PREFIX)/bin
DESTDIR ?=

VERSION_FILE := VERSION
VERSION      ?=
STRIP_LD     ?= -s -w

.PHONY: build run tidy clean fmt vet install uninstall test help version

help:
	@echo "fleet — built with the tui-cli-pattern skill"
	@echo ""
	@echo "Common targets:"
	@echo "  make            build the binary in this directory ($(BIN))"
	@echo "  make run        build then launch the binary"
	@echo "  make install    bump patch in VERSION, build, install"
	@echo "                  (set explicit version: make install VERSION=1.0.0)"
	@echo "                  (install elsewhere:   make install PREFIX=\$$HOME/.local)"
	@echo "  make uninstall  remove the installed binary"
	@echo "  make version    print the current version from $(VERSION_FILE)"
	@echo "  make test       run go test ./..."
	@echo "  make fmt        gofmt -w ."
	@echo "  make vet        go vet ./..."
	@echo "  make tidy       go mod tidy"
	@echo "  make clean      remove the local build artifact"

build:
	@v="$$(grep -oE '[0-9]+\.[0-9]+\.[0-9]+' $(VERSION_FILE) 2>/dev/null | tail -n1)"; \
	[ -n "$$v" ] || v=dev; \
	echo "Building $(BIN) v$$v..."; \
	go build -ldflags '$(STRIP_LD) -X github.com/dutraph/repofleet/internal/version.Version='"$$v" -o $(BIN) $(PKG)

run: build
	./$(BIN)

install:
	@current="$$(grep -oE '[0-9]+\.[0-9]+\.[0-9]+' $(VERSION_FILE) 2>/dev/null | tail -n1)"; \
	[ -n "$$current" ] || current="0.0.0"; \
	if [ -n "$(VERSION)" ]; then \
		new="$(VERSION)"; \
		echo "Setting version: $$current -> $$new" >&2; \
	else \
		new=$$(echo "$$current" | awk -F. '{ \
			if (NF != 3) { print "ERR" } \
			else { printf "%d.%d.%d", $$1, $$2, $$3 + 1 } \
		}'); \
		if [ "$$new" = "ERR" ]; then \
			echo "ERROR: $(VERSION_FILE) has an unexpected value ($$current). Expected MAJOR.MINOR.PATCH." >&2; \
			echo "Fix it manually, then re-run, e.g. \`echo 0.1.0 > $(VERSION_FILE)\`." >&2; \
			exit 1; \
		fi; \
		echo "Bumping patch:  $$current -> $$new" >&2; \
	fi; \
	printf '%s\n' "$$new" > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE); \
	echo "Building $(BIN) v$$new..." >&2; \
	go build -ldflags '$(STRIP_LD) -X github.com/dutraph/repofleet/internal/version.Version='"$$new" -o $(BIN) $(PKG); \
	install -d $(DESTDIR)$(BINDIR); \
	install -m 0755 $(BIN) $(DESTDIR)$(BINDIR)/$(BIN); \
	echo "" >&2; \
	echo "Installed $(BIN) v$$new -> $(DESTDIR)$(BINDIR)/$(BIN)" >&2; \
	echo "Reminder: commit $(VERSION_FILE) so the bump sticks in git." >&2

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BIN)
	@echo "Removed $(DESTDIR)$(BINDIR)/$(BIN)"

version:
	@v="$$(grep -oE '[0-9]+\.[0-9]+\.[0-9]+' $(VERSION_FILE) 2>/dev/null | tail -n1)"; \
	if [ -z "$$v" ]; then \
		echo "no $(VERSION_FILE) file yet (run \`echo 0.1.0 > $(VERSION_FILE)\`)"; \
	else \
		echo "$$v"; \
	fi

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f $(BIN)
