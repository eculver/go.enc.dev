DEFAULT_TARGET: gen

# VANGEN=$(GOPATH)/bin/vangen
VANGEN=./vangen
CONFIG=vangen.json
BUILD_DIR=build

$(VANGEN):
	# go install 4d63.com/vangen
	curl -sSL https://github.com/leighmcculloch/vangen/releases/download/v1.1.3/vangen_1.1.3_linux_amd64.tar.gz | tar xz -C . vangen
	chmod +x $@

$(CONFIG):
	go run cmd/gen/*.go --domain "go.enc.dev" --vcs "github.com/eculver" --match '^go\-'> $@

$(BUILD_DIR): $(VANGEN) $(CONFIG)
	$(VANGEN) -out $(BUILD_DIR)

.PHONY: gen
gen: $(BUILD_DIR)

.PHONY: clean
clean:
	-rm $(CONFIG)
	-rm -rf $(BUILD_DIR)
