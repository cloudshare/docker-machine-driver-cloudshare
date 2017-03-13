PLATFORMS := linux/amd64 darwin/amd64 linux/386 darwin/386 windows/amd64/.exe windows/386/.exe

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))
ext = $(word 3, $(temp))
base = docker-machine-driver-cloudshare$(ext)
out_dir = dist/$(os)/$(arch)
out_file = $(out_dir)/$(base)
main := cloudshare/main.go
current_dir := $(shell pwd)


build:
	mkdir -p dist
	go build -o dist/$(base) $(main)

package: $(PLATFORMS)

windows: windows/386/.exe windows/amd64/.exe

$(PLATFORMS):
	mkdir -p dist
	GOOS=$(os) GOARCH=$(arch) go build -o $(out_file) $(main)
	cd $(out_dir); tar czf $(current_dir)/dist/$(base)_$(arch)-$(os).tar.gz $(base)

upload: package
	github-release cloudshare/docker-machine-driver-cloudshare $(TAG) master '' 'dist/*.gz'

clean:
	rm -rf dist

.PHONY: package $(PLATFORMS) build clean
