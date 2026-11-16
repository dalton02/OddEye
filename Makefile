TAGS := webkit2_41,nosqlite,purego_utp
BIN  := build/bin/oddeye

.PHONY: run build dev test

# make run: builda o app (frontend + Go) e abre o binário
run: build
	./$(BIN)

build:
	wails build -tags $(TAGS)

# make dev: modo de desenvolvimento do Wails, com recarga do frontend
dev:
	wails dev -tags $(TAGS)

test:
	go test ./...
	cd frontend && npm run check && npm test
