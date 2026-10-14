.PHONY: dev get post put patch delete

demo:
	go run cmd/demo/main.go

get:
	curl -v http://localhost:8080

echo:
	curl -v -X POST http://localhost:8080/demo/echo \
	-d 'hey'

