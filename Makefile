.PHONY: run test web serve

run:
	go run .

test:
	go test ./...

# Builds the browser version into web/.
web:
	GOOS=js GOARCH=wasm go build -o web/tide-keeper.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

serve: web
	cd web && python3 -m http.server 8080
