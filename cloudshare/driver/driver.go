package driver

/*
TODO:
	- Add region support (fetch by name, default to Miami)
	- Add project ID support (currently always created in first project of account)
	- CPU/RAM config

*/

import (
	"fmt"
	"os"

	cs "github.com/cloudshare/go-sdk/cloudshare"
	"github.com/davecgh/go-spew/spew"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/state"
)

const driverName = "cloudshare"
const defaultDockerTemplateID = "VMBl4EQ2tgOXR51HZooN9FWA2"
const envCreateTimeoutSeconds = 600
const miamiRegionID = "REKolD1-ab84YIxODeMGob9A2"

func debug(format string, args ...interface{}) {
	if os.Getenv("DEBUG") != "true" {
		return
	}
	spew.Printf(format+"\n", args...)
}

// Driver is the driver used when no driver is selected. It is used to
// connect to existing Docker hosts by specifying the URL of the host as
// an option.
type Driver struct {
	*drivers.BaseDriver
	VMTemplateID string
	VMID         string
	APIID        string
	APIKey       string
	RegionID     string
	EnvID        string
}

func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
}

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	debug("GetCreateFlags: %+v", *d)
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:  "cloudshare-vm-template",
			Usage: "VM Template ID",
			Value: defaultDockerTemplateID,
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
			Name:  "cloudshare-region-id",
			Usage: "CloudShare region ID",
			Value: miamiRegionID,
		},
	}
}

func formatEnvName(machineName string) string {
	return "docker-machine-" + machineName
}

func (d *Driver) getClient() *cs.Client {
	return &cs.Client{
		APIKey: d.APIKey,
		APIID:  d.APIID,
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

func (d *Driver) createEnv(templateID string, name string) (*string, error) {
	c := d.getClient()

	projID, apierr := getFirstProjectID(c)
	if apierr != nil {
		return nil, apierr
	}

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
		return nil, apierr
	}

	envID := envCreateResponse.EnvironmentID
	d.EnvID = envID

	return &envID, nil
}

func (d *Driver) Create() error {
	debug("Create: Driver %+v", *d)

	envName := formatEnvName(d.BaseDriver.MachineName)
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

	d.VMID = *envID
	return nil
}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	debug("DriverName: Driver %+v", *d)
	return driverName
}

func (d *Driver) GetIP() (string, error) {
	debug("GetIP: Driver %+v", *d)
	return d.IPAddress, nil
}

func (d *Driver) GetSSHHostname() (string, error) {
	debug("GetSSHHostname: Driver %+v", *d)
	return "", nil
}

func (d *Driver) GetSSHKeyPath() string {
	debug("GetSSHKeyPath: Driver %+v", *d)
	return ""
}

func (d *Driver) GetSSHPort() (int, error) {
	debug("GetSSHPort: Driver %+v", *d)
	return 0, nil
}

func (d *Driver) GetSSHUsername() string {
	debug("GetSSHUsername: Driver %+v", *d)
	return ""
}

func (d *Driver) GetURL() (string, error) {
	// TODO: find env by name, fetch vm public DNs
	debug("GetURL: Driver %+v", *d)
	return "no url yet", nil
}

func (d *Driver) getEnvStatus(envID string) (cs.EnvironmentStatusCode, error) {
	env := cs.EnvironmentExtended{}
	err := d.getClient().GetEnvironmentExtended(envID, &env)
	return env.StatusCode, err
}

func (d *Driver) GetState() (state state.State, err error) {
	debug("GetState: Driver %+v", *d)
	status, err := d.getEnvStatus(d.EnvID)
	state = ToDockerMachineState(status)
	debug("Current state: %d = %d", status, state)
	return
}

func (d *Driver) Kill() error {
	debug("Kill: Driver %+v", *d)
	return fmt.Errorf("hosts without a driver cannot be killed")
}

func (d *Driver) Remove() error {
	debug("Remove: Driver %+v", *d)
	return nil
}

func (d *Driver) Restart() error {
	debug("Restart: Driver %+v", *d)
	return fmt.Errorf("hosts without a driver cannot be restarted")
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
	debug("SetConfigFromFlags: Driver %+v\nflags: %+v", *d, flags)
	templateID := flags.String("cloudshare-vm-template")

	if err := validateRequired([]string{"cloudshare-api-key",
		"cloudshare-api-id", "cloudshare-region-id"}, flags); err != nil {
		return err
	}

	d.VMTemplateID = templateID
	d.APIID = flags.String("cloudshare-api-id")
	d.APIKey = flags.String("cloudshare-api-key")
	d.RegionID = flags.String("cloudshare-region-id")
	return nil
}

func (d *Driver) Start() error {
	debug("Start: Driver %+v", *d)
	return fmt.Errorf("hosts without a driver cannot be started")
}

func (d *Driver) Stop() error {
	debug("Stop: Driver %+v", *d)
	return fmt.Errorf("hosts without a driver cannot be stopped")
}
