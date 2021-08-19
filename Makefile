build:
	go build -o bin/datahub-config-deployment cmd/deploy/main.go

run:
	go run cmd/deploy/main.go

docker:
	docker build . -t datahub-config-deployment

mim-deploy:
	go build -o bin/mim-deploy ./cmd/deploy/main.go
