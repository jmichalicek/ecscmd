
build: build-linux-amd64 build-osx-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o .build/linux-amd64/ecscmd .

build-osx-amd64:
	GOOS=darwin GOARCH=amd64 go build -o .build/osx-amd64/ecscmd .

docker-alpine:
	docker build -f Dockerfile.alpine -t jmichalicek/ecscmd:alpine .
	docker tag jmichalicek/ecscmd:alpine jmichalicek/ecscmd:alpine-3.7
	docker tag jmichalicek/ecscmd:alpine jmichalicek/ecscmd:alpine-latest

push-docker-alpine:
	docker push jmichalicek/ecscmd:alpine
	docker push jmichalicek/ecscmd:alpine-3.7
