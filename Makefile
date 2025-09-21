ssh:
	echo 'password in mail or reset'
	ssh-copy-id root@45.80.69.213
	ssh root@45.80.69.213

up:
	docker compose -f ./docs/postgres/docker-compose.yml up -d

set-context:
	echo 'bla'

deploy:
	GOOS=linux GOARCH=amd64 go build -a -o ./build/app main.go
	DOCKER_HOST="ssh://root@45.80.69.213" docker compose up --build -d

deploy-down:
	DOCKER_HOST="ssh://root@45.80.69.213" docker compose down

down:
	docker compose -f ./docs/postgres/docker-compose.yml down

reverse-proxy:
	~/telebit http 8081

format:
	@echo 'formatting'
	@go fmt ./...
	@gci write -s standard -s default -s localmodule --skip-generated --skip-vendor .

build:
	GOOS=linux GOARCH=amd64 go build -a -o ./build/app main.go

test:
	go test ./...

install-tools:
	@go install github.com/daixiang0/gci@latest