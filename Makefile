b64: *.go
	go build -o b64 .

.PHONY:test
test: b64
	bash ./example.sh
