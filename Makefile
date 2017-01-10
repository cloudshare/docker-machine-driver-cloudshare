PLATFORMS := linux/amd64 darwin/amd64 linux/386 darwin/386

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

main = cloudshare/main.go

build:
	go build -o docker-machine-driver-cloudshare $(main)

release: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o 'docker-machine-driver-cloudshare_$(os)-$(arch)' $(main)

.PHONY: release $(PLATFORMS)
