package driver

/*
TODO:
	- Add region support (fetch by name, default to Miami)
	- Add project ID support (currently always created in first project of account)
	- CPU/RAM config
	- Improve instance state mapping: https://github.com/docker/machine/blob/master/drivers/amazonec2/amazonec2.go#L774
	- Disable debug printing of password/api-key
	- Fix cloudfolders issue
	- Add NOPASSWD: to VM templates
	- Kill?
	- Fix reboot
	- Regenerate

*/

import (
	"fmt"
	"time"

	cs "github.com/cloudshare/go-sdk/cloudshare"
	"github.com/davecgh/go-spew/spew"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	dssh "github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
)

const driverName = "cloudshare"
const docker16Template = "VMBl4EQ2tgOXR51HZooN9FWA2"
const docker14Template = "VMQ5ZA0uXzxxGyQfYdS5RxaQ2"
const envCreateTimeoutSeconds = 600
const miamiRegionID = "REKolD1-ab84YIxODeMGob9A2"
const defaultUserName = "sysadmin"
const defaultSSHPort = 22
const defaultPort = 2376

func debug(format string, args ...interface{}) {
	msg := spew.Sprintf(format+"\n", args...)
	log.Debug(msg)
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
	Hostname     string
	Password     string
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

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	debug("GetCreateFlags: %+v", *d)
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
	debug("Env create request: %+v", request)
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

	// TODO: figure out a way to avoid this on ubuntu 16
	// if err := d.sshRun("rm -rf /etc/init.d/cloudfolders"); err != nil {
	// 	return err
	// }

	return nil

}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	return driverName
}

func (d *Driver) GetIP() (string, error) {
	if err := d.verifyHostnameKnown(); err != nil {
		return "", err
	}
	return d.Hostname, nil
}

func (d *Driver) sshRun(command string) error {
	return d.sessionAction(func(session *ssh.Session) error {
		log.Debugf("Executing SSH: %s", command)
		return session.Run(command)
	})
}

func (d *Driver) sessionAction(action func(session *ssh.Session) error) error {
	client, err := d.newSSHClient()
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()
	return action(session)
}

func (d *Driver) sshCopy(localFile string, remoteFile string) error {
	return d.sessionAction(func(session *ssh.Session) error {
		return scp.CopyPath(localFile, remoteFile, session)
	})
}

func (d *Driver) newSSHClient() (*ssh.Client, error) {
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", d.Hostname, defaultSSHPort), &ssh.ClientConfig{
		User: d.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(d.Password),
		},
	})
}

func (d *Driver) installSSHCertificate() error {
	log.Info("Installing SSH certificates on new VM...")
	log.Debugf("SSH client created to %s:%s@%s:%d", d.SSHUser, d.Password, d.Hostname, defaultSSHPort)

	log.Debugf("Testing SSH connection")
	err := d.sshRun("exit 0")
	if err != nil {
		log.Errorf("Failed SSH command: %s", err)
		return err
	}

	log.Debug("Generating SSH private key...")
	if err = dssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return err
	}

	log.Debug("Copying public key to remote VM...")
	pubKeyFile := d.GetSSHKeyPath() + ".pub"
	if err = d.sshCopy(pubKeyFile, "~/.ssh/authorized_keys"); err != nil {
		return err
	}

	log.Debug("Adding public key to authorized_keys...")
	if err = d.sshRun("chmod 600 ~/.ssh/authorized_keys"); err != nil {
		return err
	}

	cmd := fmt.Sprintf("echo '%s' | sudo -S sed -i 's/^%%sudo.*$/%%sudo ALL=(ALL:ALL) NOPASSWD: ALL/' /etc/sudoers", d.Password)
	log.Debug("Granting passwordless sudo access to user...")
	if err = d.sshRun(cmd); err != nil {
		return err
	}

	return nil
}

func (d *Driver) GetSSHHostname() (string, error) {
	debug("GetSSHHostname: Driver %+v", *d)
	if err := d.verifyHostnameKnown(); err != nil {
		return "", err
	}

	return d.Hostname, nil
}

func (d *Driver) GetSSHUsername() string {
	debug("GetSSHUsername: Driver %+v", *d)
	return defaultUserName
}

func (d *Driver) formatURL() string {
	url := fmt.Sprintf("tcp://%s:2376", d.Hostname)
	debug(url)
	return url
}

func (d *Driver) verifyHostnameKnown() error {
	if d.Hostname != "" {
		return nil
	}
	status, err := d.getEnvStatus(d.EnvID)
	if err != nil {
		debug("failed to fetch env status")
		return err
	}
	if status != cs.StatusReady {
		return fmt.Errorf("machine not yet in Ready state")
	}

	extended := cs.EnvironmentExtended{}
	if err = d.getClient().GetEnvironmentExtended(d.EnvID, &extended); err != nil {
		return err
	}

	if len(extended.Vms) < 1 {
		return fmt.Errorf("environment contains no VMs")
	}
	d.Hostname = extended.Vms[0].Fqdn
	d.Password = extended.Vms[0].Password
	d.VMID = extended.Vms[0].ID
	d.SSHUser = defaultUserName
	return nil
}

func (d *Driver) GetURL() (string, error) {
	debug("GetURL: Driver %+v", *d)
	if err := d.verifyHostnameKnown(); err != nil {
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
		extended := cs.EnvironmentExtended{}
		err := d.getClient().GetEnvironmentExtended(d.EnvID, &extended)
		if err != nil {
			return err
		}
		log.Debugf("VM status: %s", extended.Vms[0].StatusText)
		time.Sleep(time.Second * 3)
	}
	return nil
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
	log.Infof("Resuming environment %s", d.EnvID)
	return d.getClient().EnvironmentResume(d.EnvID)
}

func (d *Driver) Stop() error {
	log.Infof("Suspending environment %s", d.EnvID)
	return d.getClient().EnvironmentSuspend(d.EnvID)
}
