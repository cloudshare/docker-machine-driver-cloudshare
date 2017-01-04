package main

import (
	driver "github.com/cloudshare/docker-machine-driver-cloudshare/cloudshare/driver"
	"github.com/docker/machine/libmachine/drivers/plugin"
)

func main() {
	plugin.RegisterDriver(driver.NewDriver("", ""))
}
