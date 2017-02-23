package driver

import (
	"fmt"
	"time"

	cs "github.com/cloudshare/go-sdk/cloudshare"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/state"
)

const driverName = "cloudshare"
const docker16Template = "Docker - Ubuntu 16.04 Server"
const docker14Template = "Docker - Ubuntu 14.04 Server - SMALL"
const envCreateTimeoutSeconds = 300
const envAdjustTimeoutSeconds = 600
const defaultRegion = "Miami"
const defaultUserName = "sysadmin"
const defaultSSHPort = 22
const defaultPort = 2376

// You can grab this map from api/v3/regions, but since regions change not very frequently there's no reason to execute this API call each time we create a machine...
var regions = map[string]string{
	"Miami":            "REKolD1-ab84YIxODeMGob9A2",
	"VMware_Singapore": "RE0YOUV7_lTmgb0X8D1UjM3g2",
	"VMWare_Amsterdam": "RE6OEZs-y-mkK1mEMGwIgZiw2",
}

// Driver is the driver used when no driver is selected. It is used to
// connect to existing Docker hosts by specifying the URL of the host as
// an option.
type Driver struct {
	*drivers.BaseDriver

	CPUs         int
	DiskSizeGB   int
	ExpiryDays   int
	MemorySizeMB int

	APIID          string
	APIKey         string
	EnvID          string
	Hostname       string
	Password       string
	ProjectID      string
	RegionID       string
	VMID           string
	VMTemplateID   string
	VMTemplateName string
}

func NewDriver(hostName, storePath string) *Driver {
	d := &Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
	return d
}

func formatEnvName(machineName string) string {
	return "docker-machine-" + machineName
}

func (d *Driver) getClient() *cs.Client {
	return &cs.Client{
		APIKey: d.APIKey,
		APIID:  d.APIID,
		Tags:   "docker-machine-driver",
	}
}

func getFirstProjectID(c *cs.Client) (*string, error) {
	projects := []cs.Project{}
	apierr := c.GetProjects(&projects)
	if apierr != nil {
		return nil, apierr
	}
	if len(projects) < 1 {
		panic("User account contains no projects")
	}
	return &projects[0].ID, nil
}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	return driverName
}

func (d *Driver) GetIP() (string, error) {
	if _, err := d.verifyHostnameKnown(); err != nil {
		return "", err
	}
	return d.Hostname, nil
}

func (d *Driver) formatURL() string {
	url := fmt.Sprintf("tcp://%s:2376", d.Hostname)
	return url
}

func (d *Driver) verifyHostnameKnown() (*cs.EnvironmentExtended, error) {
	if d.EnvID == "" {
		return nil, fmt.Errorf("Environment ID not yet configured. Is the machine still being created?")
	}

	extended := cs.EnvironmentExtended{}
	if err := d.getClient().GetEnvironmentExtended(d.EnvID, &extended); err != nil {
		return nil, err
	}

	if extended.StatusCode != cs.StatusReady {
		return nil, fmt.Errorf("env status %s. Waiting for Running", extended.StatusText)
	}

	if len(extended.Vms) < 1 {
		return &extended, fmt.Errorf("environment contains no VMs")
	}

	log.Debugf("VM state: %s", extended.Vms[0].StatusText)
	if extended.Vms[0].StatusText != "Running" {
		return &extended, fmt.Errorf("VM not yet running. Current state: %s", extended.Vms[0].StatusText)
	}
	if d.Hostname == "" {
		d.Hostname = extended.Vms[0].Fqdn
		d.Password = extended.Vms[0].Password
		d.VMID = extended.Vms[0].ID
		d.SSHUser = defaultUserName
	}
	return &extended, nil
}

func (d *Driver) GetURL() (string, error) {
	if _, err := d.verifyHostnameKnown(); err != nil {
		return "", err
	}
	return d.formatURL(), nil
}

func (d *Driver) getEnvStatus(envID string) (cs.EnvironmentStatusCode, error) {
	env := cs.EnvironmentExtended{}
	err := d.getClient().GetEnvironmentExtended(envID, &env)
	return env.StatusCode, err
}

func (d *Driver) GetState() (state state.State, err error) {
	status, err := d.getEnvStatus(d.EnvID)
	state = ToDockerMachineState(status)
	return
}

func (d *Driver) Kill() error {
	return fmt.Errorf("kill is not supported for CloudShare docker machines. You can stop, rm or restart")
}

func (d *Driver) Remove() error {
	log.Infof("Deleting environment %s", d.EnvID)
	return d.getClient().EnvironmentDelete(d.EnvID)
}

func (d *Driver) Restart() error {
	log.Infof("Rebooting VM %s...", d.VMID)
	if err := d.getClient().RebootVM(d.VMID); err != nil {
		log.Errorf("Error rebooting VM %s: %s", d.VMID, err)
		return err
	}
	for {
		time.Sleep(time.Second * 3)
		extended := cs.EnvironmentExtended{}
		err := d.getClient().GetEnvironmentExtended(d.EnvID, &extended)
		if err != nil {
			return err
		}
		status := extended.Vms[0].StatusText
		log.Debugf("VM status: %s", status)
		if status != "Running" && status != "Rebooting" {
			return fmt.Errorf("Unexpected VM status: %s", status)
		}
		if status == "Running" {
			break
		}
	}
	log.Info("VM rebooted")
	return nil
}

func (d *Driver) Start() error {
	log.Infof("Resuming environment %s", d.EnvID)
	return d.getClient().EnvironmentResume(d.EnvID)
}

func (d *Driver) Stop() error {
	log.Infof("Suspending environment %s", d.EnvID)
	return d.getClient().EnvironmentSuspend(d.EnvID)
}
