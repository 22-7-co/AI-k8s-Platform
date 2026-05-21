.PHONY: help build test clean operator exporter

BINARY_OPERATOR := bin/operator
BINARY_EXPORTER := bin/exporter

help:
	@echo "Targets:"
	@echo "  build     - build operator and exporter"
	@echo "  operator  - build operator only"
	@echo "  exporter  - build exporter only"
	@echo "  test      - go test ./..."
	@echo "  clean     - remove bin/"

build: operator exporter

operator:
	@mkdir -p bin
	go build -o $(BINARY_OPERATOR) ./cmd/operator

exporter:
	@mkdir -p bin
	go build -o $(BINARY_EXPORTER) ./cmd/exporter

test:
	go test ./...

clean:
	rm -rf bin/
