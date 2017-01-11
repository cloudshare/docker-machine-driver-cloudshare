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
	return nil
}
