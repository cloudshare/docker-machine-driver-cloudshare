package driver

import (
	"fmt"
	cs "github.com/cloudshare/go-sdk/cloudshare"
	"github.com/docker/machine/libmachine/log"
	"time"
)

func (d *Driver) createEnv(templateID string, name string) (*string, error) {
	log.Infof("Creating a new Environment based on the VM template (%s)...", templateID)
	c := d.getClient()

	projID, apierr := getFirstProjectID(c)
	if apierr != nil {
		return nil, apierr
	}
	log.Debugf("Project ID: %s", *projID)

	request := cs.EnvironmentTemplateRequest{
		Environment: cs.Environment{
			Name:        name,
			Description: "Docker-Machine Environment",
			ProjectID:   *projID,
			RegionID:    d.RegionID,
		},
		ItemsCart: []cs.VM{{
			Type:         2,
			Name:         "docker-machine",
			TemplateVMID: d.VMTemplateID,
			Description:  "Docker-Machine VM",
		}},
	}
	envCreateResponse := cs.CreateTemplateEnvResponse{}

	apierr = c.EnvironmentCreateFromTemplate(&request, &envCreateResponse)

	if apierr != nil {
		log.Errorf("Failed to create env")
		return nil, apierr
	}

	envID := envCreateResponse.EnvironmentID
	d.EnvID = envID
	log.Infof("CloudShare environment created: %s", d.EnvID)

	return &envID, nil
}

func (d *Driver) Create() error {
	envName := formatEnvName(d.BaseDriver.MachineName)
	log.Debugf("Creating environment %s...", envName)
	c := d.getClient()

	env, apierr := c.GetEnvironmentByName(envName)
	if apierr != nil {
		return apierr
	}

	if env != nil {
		return fmt.Errorf("Docker-Machine enviroment already exists for '%s': %s", d.BaseDriver.MachineName, env.URL())
	}
	envID, apierr := d.createEnv(d.VMTemplateID, envName)
	if apierr != nil {
		return apierr
	}

	d.EnvID = *envID

	// Wait for ready state and set hostname
	log.Info("Waiting for new environment to become Ready...")

	var pollInterval time.Duration = 5
	for i := time.Duration(5); i < envCreateTimeoutSeconds; i += pollInterval {
		time.Sleep(time.Second * pollInterval)
		if err := d.verifyHostnameKnown(); err == nil {
			break
		}
		log.Debugf("Still waiting for hostname...")
	}
	if err := d.verifyHostnameKnown(); err != nil {
		return err
	}

	if err := d.installSSHCertificate(); err != nil {
		return err
	}

	return nil
}
