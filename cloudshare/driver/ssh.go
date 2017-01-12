package driver

import "fmt"
import dssh "github.com/docker/machine/libmachine/ssh"
import "golang.org/x/crypto/ssh"
import "github.com/tmc/scp"
import "github.com/docker/machine/libmachine/log"

func (d *Driver) GetSSHUsername() string {
	return defaultUserName
}

func (d *Driver) sshRun(command string) error {
	return d.sessionAction(func(session *ssh.Session) error {
		log.Debugf("Executing SSH: %s", command)
		return session.Run(command)
	})
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

	return nil
}

func (d *Driver) GetSSHHostname() (string, error) {
	if _, err := d.verifyHostnameKnown(); err != nil {
		return "", err
	}

	return d.Hostname, nil
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
