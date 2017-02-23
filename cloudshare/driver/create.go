package driver

import (
	"fmt"
	cs "github.com/cloudshare/go-sdk/cloudshare"
	"github.com/docker/machine/libmachine/log"
	"time"
)

func (d *Driver) verifyPolicyExists(projID string, expiryDays int, policyID *string) error {
	policyName := fmt.Sprintf("docker-machine-policy-delete-after-%d-days", expiryDays)
	log.Debugf("Searching for existing policy '%s'", policyName)
	c := d.getClient()
	existingPolicies := []cs.Policy{}
	if apierr := c.GetPolicies(projID, &existingPolicies); apierr != nil {
		return apierr
	}
	for _, existingPolicy := range existingPolicies {
		if existingPolicy.Name == policyName {
			*policyID = existingPolicy.ID
			log.Debug("Policy found, no need to create one.")
			return nil
		}
	}

	request := cs.PolicyRequest{
		Name:                                 policyName,
		ProjectID:                            projID,
		RunTimeTotalMinutes:                  expiryDays * 24 * 60,
		DiskTimeTotalMinutes:                 expiryDays * 24 * 60,
		AutoAction:                           "SuspendTheEnvironment",
		AutoActionThresholdMinutesForSuspend: 15,
	}

	response := cs.PolicyCreationResponse{}
	log.Debugf("Creating new policy with spec %+v", request)
	if apierr := c.CreateProjectPolicy(request, &response); apierr != nil {
		return apierr
	}
	*policyID = response.ID
	log.Debugf("Policy created (id: %s)", policyID)
	return nil
}

func (d *Driver) createEnv(templateID string, name string) (*string, error) {
	log.Infof("Creating a new Environment based on the VM template (%s)...", templateID)
	c := d.getClient()

	projID := d.ProjectID
	var apierr error

	if projID == "" {
		log.Debugf("Project ID not specified. Querying for account's first project.")
		id, apierr := getFirstProjectID(c)
		if apierr != nil {
			return nil, apierr
		}
		projID = *id
	}

	log.Debugf("Project ID: %s", projID)

	var policyID string
	if d.ExpiryDays != 0 {
		if apierr := d.verifyPolicyExists(projID, d.ExpiryDays, &policyID); apierr != nil {
			return nil, apierr
		}
	}

	env := cs.Environment{
		Name:        name,
		Description: "Docker-Machine Environment",
		ProjectID:   projID,
		RegionID:    d.RegionID,
		PolicyID:    policyID,
	}

	if policyID != "" {
		env.PolicyID = policyID
	}

	request := cs.EnvironmentTemplateRequest{
		Environment: env,
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

func (d *Driver) adjustHW() error {
	c := d.getClient()

	extended := cs.EnvironmentExtended{}
	if err := c.GetEnvironmentExtended(d.EnvID, &extended); err != nil {
		return err
	}

	request := cs.EditVMHardwareRequest{
		VMID: d.VMID,
	}

	currentCPUs := extended.Vms[0].CPUCount
	anyMods := false
	if d.CPUs != 0 && currentCPUs != d.CPUs {
		log.Infof("Adjusting number of CPUs from %d to %d", currentCPUs, d.CPUs)
		request.NumCPUs = d.CPUs
		anyMods = true
	}

	currentRAM := extended.Vms[0].MemorySizeMB
	if d.MemorySizeMB != 0 && currentRAM != d.MemorySizeMB {
		log.Infof("Adjusting VM memory from %d MBs to %d MBs", currentRAM, d.MemorySizeMB)
		request.MemorySizeMBs = d.MemorySizeMB
		anyMods = true
	}

	currentDisk := extended.Vms[0].DiskSizeGB
	if d.DiskSizeGB != 0 && currentDisk != d.DiskSizeGB {
		if d.DiskSizeGB < currentDisk {
			return fmt.Errorf("Requested disk size cannot be smaller than %d GBs", currentDisk)
		}
		log.Infof("Adjusting disk size from %d GBs to %d GBs", currentDisk, d.DiskSizeGB)
		request.DiskSizeGBs = d.DiskSizeGB
		anyMods = true
	}

	if anyMods {
		response := cs.EditVMHardwareResponse{}
		if err := c.EditVMHardware(request, &response); err != nil {
			return err
		}

		log.Info("Waiting for adjusted VM to become ready...")
		if err := d.waitForReady(envAdjustTimeoutSeconds); err != nil {
			return err
		}
	}

	return nil
}

func (d *Driver) waitForReady(timeoutSeconds time.Duration) error {

	var pollInterval time.Duration = 10
	for i := pollInterval; i < timeoutSeconds; i += pollInterval {
		time.Sleep(time.Second * pollInterval)
		if _, err := d.verifyHostnameKnown(); err != nil {
			log.Debugf("Still waiting for env to be ready... %v", err)
		} else {
			break
		}

	}
	return nil
}

func (d *Driver) resolveTemplateID() (string, error) {
	log.Debugf("Searching for template with name %s...", d.VMTemplateName)
	c := d.getClient()
	templParams := cs.GetTemplateParams{
		TemplateType: "1",
	}
	templates := []cs.VMTemplate{}
	if err := c.GetTemplates(&templParams, &templates); err != nil {
		return "", err
	}
	for _, template := range templates {
		if template.Name == d.VMTemplateName {
			return template.ID, nil
		}
	}
	return "", fmt.Errorf("template with name '%s' not found", d.VMTemplateName)
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

	if d.VMTemplateID == "" {
		d.VMTemplateID, apierr = d.resolveTemplateID()
		if apierr != nil {
			return apierr
		}
	}

	envID, apierr := d.createEnv(d.VMTemplateID, envName)
	if apierr != nil {
		return apierr
	}

	d.EnvID = *envID

	// Wait for ready state and set hostname
	log.Info("Waiting for new environment to become Ready...")
	if err := d.waitForReady(envCreateTimeoutSeconds); err != nil {
		return err
	}

	if _, err := d.verifyHostnameKnown(); err != nil {
		return err
	}

	if err := d.adjustHW(); err != nil {
		return err
	}

	if err := d.installSSHCertificate(); err != nil {
		return err
	}

	return nil
}
