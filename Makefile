DEFAULT_TARGET: gen

BUILD_DIR=build

$(BUILD_DIR):
	go run cmd/gen/*.go --domain "go.enc.dev" --vcs "github.com/eculver" --match '^go\-' --output-dir $(BUILD_DIR) --template-dir ./tmpl

.PHONY: gen
gen: $(BUILD_DIR)

.PHONY: clean
clean:
	-rm -rf $(BUILD_DIR)
