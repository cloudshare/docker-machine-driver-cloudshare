.PHONY: build

build:
	go build -i -o docker-machine-driver-cloudshare cloudshare/main.go
