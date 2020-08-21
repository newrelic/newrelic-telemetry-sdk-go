#
# Makefile fragment for installing deps
#

GO           ?= go
GOFMT        ?= gofmt
VENDOR_CMD   ?= ${GO} mod tidy
BUILD_DIR    ?= ./bin/

# Go file to track tool deps with go modules
TOOL_DIR     ?= tools
TOOL_CONFIG  ?= $(TOOL_DIR)/tools.go

# These should be mirrored in /tools.go to keep versions consistent
GOTOOLS      += github.com/client9/misspell/cmd/misspell


tools: check-version tools-compile
	@echo "=== $(PROJECT_NAME) === [ tools            ]: Installing tools required by the project..."
	@cd $(TOOL_DIR)
	@$(GO) install $(GOTOOLS)
	@$(VENDOR_CMD)


tools-update: check-version
	@echo "=== $(PROJECT_NAME) === [ tools-update     ]: Updating tools required by the project..."
	@cd $(TOOL_DIR)
	@$(GO) get -u $(GOTOOLS)
	@$(VENDOR_CMD)

tools-config: tools
	@echo "=== $(PROJECT_NAME) === [ tools-config     ]: Updating tool configuration $(TOOL_CONFIG) ..."
	@echo "// +build tools\n\npackage tools\n\nimport (" > $(TOOL_CONFIG)
	@for t in $(GOTOOLS); do \
		echo "\t_ \"$$t\"" >> $(TOOL_CONFIG) ; \
	done
	@echo ")" >> $(TOOL_CONFIG)
	@$(GOFMT) -w $(TOOL_CONFIG)
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

deps: tools deps-only


# Determine commands by looking into cmd/*
TOOL_COMMANDS   ?= $(shell find ${SRCDIR}/tools -depth 1 -type d)
# Determine binary names by stripping out the dir names
TOOL_BINS       := $(foreach tool,${TOOL_COMMANDS},$(notdir ${tool}))

tools-compile: deps-only
	@echo "=== $(PROJECT_NAME) === [ tools-compile    ]: building custom tools:"
	@for b in $(TOOL_BINS); do \
		echo "=== $(PROJECT_NAME) === [ tools-compile    ]:     $$b"; \
		BUILD_FILES=`find $(SRCDIR)/tools/$$b -type f -name "*.go"` ; \
		GOOS=$(GOOS) $(GO) build -ldflags=$(LDFLAGS) -o $(SRCDIR)/$(BUILD_DIR)/$$b $$BUILD_FILES ; \
	done

deps-only:
	@echo "=== $(PROJECT_NAME) === [ deps             ]: Installing package dependencies required by the project..."
	@$(VENDOR_CMD)

.PHONY: deps deps-only tools tools-update
