.PHONY: build clean deploy

build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/sites/create endpoints/sites/create/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sites/delete endpoints/sites/delete/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sites/get endpoints/sites/get/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sites/list endpoints/sites/list/main.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	sls deploy --verbose
