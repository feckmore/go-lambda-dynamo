.PHONY: build clean deploy

build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/create functions/create/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/delete functions/delete/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/get functions/get/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/list functions/list/main.go

clean:
	rm -rf .functions/bin ./vendor Gopkg.lock

deploy: clean build
	sls deploy --verbose
