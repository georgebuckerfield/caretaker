NAME ?= caretaker

container: build-linux
	docker build .

build:
	go build -o bin/$(NAME)

build-linux:
	env GOOS=linux go build -o bin/$(NAME)
