DEFAULT_TARGET: gen

VANGEN=$(GOPATH)/bin/vangen
CONFIG=vangen.json
BUILD_DIR=build

$(VANGEN):
	go get 4d63.com/vangen

$(CONFIG):
	go run cmd/gen/*.go --domain "go.enc.dev" --vcs "github.com/eculver" --match '^go\-'> $@

$(BUILD_DIR): $(VANGEN) $(CONFIG)
	vangen -out $(BUILD_DIR)

.PHONY: gen
gen: $(BUILD_DIR)

.PHONY: clean
clean:
	-rm $(CONFIG)
	-rm -rf $(BUILD_DIR)
