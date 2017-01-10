PLATFORMS := linux/amd64 darwin/amd64 linux/386 darwin/386

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))
base := docker-machine-driver-cloudshare
out_dir = dist/$(os)/$(arch)
out_file = $(out_dir)/$(base)
main := cloudshare/main.go
current_dir := $(shell pwd)

build:
	mkdir -p dist
	go build -o $(base) $(main)

release: $(PLATFORMS)

$(PLATFORMS):
	mkdir -p dist
	GOOS=$(os) GOARCH=$(arch) go build -o $(out_file) $(main)
	cd $(out_dir); tar czf $(current_dir)/dist/$(base)_$(arch)-$(os).tar.gz $(base)


clean: 
	rm -rf dist

.PHONY: release $(PLATFORMS) build clean
