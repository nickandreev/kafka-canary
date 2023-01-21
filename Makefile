.PHONY build:
build:
	go build -v ./cmd/kafka-canary

.PHONY test:
test:
	go test -v -cover -race -parallel 	./...

.PHONY fmt:
fmt:
	gofmt -l -s -w ./
	goimports -l --local "github.com/pecigonzalo/kafka-canary" -w ./