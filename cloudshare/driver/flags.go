package driver

import (
	"fmt"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
)

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:  "cloudshare-vm-template-id",
			Usage: "VM Template ID",
		},
		mcnflag.StringFlag{
			Name:  "cloudshare-vm-template-name",
			Usage: "VM Template name (exact name; cannot be used with --cloudshare-vm-template-id)",
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

	if err := validateRequired([]string{"cloudshare-api-key",
		"cloudshare-api-id", "cloudshare-region-name"}, flags); err != nil {
		return err
	}

	templateID := flags.String("cloudshare-vm-template-id")
	templateName := flags.String("cloudshare-vm-template-name")
	if templateID == "" && templateName == "" {
		return fmt.Errorf("either cloudshare-vm-template-id or cloudshare-vm-template-name must be specified")
	}

	d.VMTemplateID = templateID
	d.VMTemplateName = templateName
	d.APIID = flags.String("cloudshare-api-id")
	d.APIKey = flags.String("cloudshare-api-key")
	d.RegionID = regions[flags.String("cloudshare-region-name")]
	d.CPUs = flags.Int("cloudshare-cpus")
	d.MemorySizeMB = flags.Int("cloudshare-ram-mb")
	d.DiskSizeGB = flags.Int("cloudshare-disk-gb")
	return nil
}
