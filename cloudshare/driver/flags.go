package driver

import (
	"fmt"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
)

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:  "cloudshare-vm-template",
			Usage: "VM Template ID",
			Value: docker14Template,
		},
		mcnflag.StringFlag{
			Name:   "cloudshare-api-id",
			Usage:  "CloudShare API ID (required)",
			EnvVar: "CLOUDSHARE_API_ID",
		},
		mcnflag.StringFlag{
			Name:   "cloudshare-api-key",
			Usage:  "CloudShare API KEY (required)",
			EnvVar: "CLOUDSHARE_API_KEY",
		},
		mcnflag.StringFlag{
			Name:  "cloudshare-region-name",
			Usage: "CloudShare region name",
			Value: "Miami",
		},
		mcnflag.IntFlag{
			Name:  "cloudshare-disk-gb",
			Usage: "Disk size (GB), >=10GB",
			Value: 10,
		},
		mcnflag.IntFlag{
			Name:  "cloudshare-cpus",
			Usage: "CPU count",
			Value: 1,
		},
		mcnflag.IntFlag{
			Name:  "cloudshare-ram-mb",
			Usage: "RAM (MBs) 256-32768",
			Value: 2048,
		},
	}
}

func validateRequired(requiredFlags []string, flags drivers.DriverOptions) error {
	for _, req := range requiredFlags {
		value := flags.String(req)
		if value == "" {
			return fmt.Errorf("%s is a required field", req)
		}
	}
	return nil
}

func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	templateID := flags.String("cloudshare-vm-template")

	if err := validateRequired([]string{"cloudshare-api-key",
		"cloudshare-api-id", "cloudshare-region-name"}, flags); err != nil {
		return err
	}

	d.VMTemplateID = templateID
	d.APIID = flags.String("cloudshare-api-id")
	d.APIKey = flags.String("cloudshare-api-key")
	d.RegionID = regions[flags.String("cloudshare-region-name")]
	d.CPUs = flags.Int("cloudshare-cpus")
	d.MemorySizeMB = flags.Int("cloudshare-ram-mb")
	d.DiskSizeGB = flags.Int("cloudshare-disk-gb")
	return nil
}
