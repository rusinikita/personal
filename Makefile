# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

ssh:
	echo 'password in mail or reset'
	ssh-copy-id $(SSH_ACCESS)
	ssh $(SSH_ACCESS)

up:
	docker compose -f ./docs/postgres/docker-compose.yml up -d

set-context:
	echo 'bla'

deploy:
	GOOS=linux GOARCH=amd64 go build -a -o ./build/app main.go
	DOCKER_HOST="ssh://$(SSH_ACCESS)" docker compose up --build -d

deploy-down:
	DOCKER_HOST="ssh://$(SSH_ACCESS)" docker compose down

down:
	docker compose -f ./docs/postgres/docker-compose.yml down

reverse-proxy:
	~/telebit http 8881

format:
	@echo 'formatting'
	@go fmt ./...
	@gci write -s standard -s default -s localmodule --skip-generated --skip-vendor .

build-app:
	GOOS=linux GOARCH=amd64 go build -a -o ./build/app main.go

test:
	go test ./...

install-tools:
	@go install github.com/daixiang0/gci@latest