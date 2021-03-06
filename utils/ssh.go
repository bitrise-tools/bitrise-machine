package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/bitrise-machine/config"
)

// StartAsyncCommandThroughSSHWithWriters ...
func StartAsyncCommandThroughSSHWithWriters(sshConfigModel config.SSHConfigModel, cmdToRunWithSSH string, stdout, stderr io.Writer) (*exec.Cmd, error) {
	sshArgs := sshConfigModel.SSHCommandArgs()
	fullArgs := append(sshArgs, cmdToRunWithSSH)

	cmd := exec.Command("ssh", fullArgs...)
	log.Debugf("StartAsyncCommandThroughSSHWithWriters: Full command to run: %v", cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd, cmd.Start()
}

// RunCommandThroughSSHWithWriters ...
// TODO: this could possibly use StartAsyncCommandThroughSSHWithWriters and `return cmd.Wait()`
func RunCommandThroughSSHWithWriters(sshConfigModel config.SSHConfigModel, cmdToRunWithSSH string, stdout, stderr io.Writer) error {
	sshArgs := sshConfigModel.SSHCommandArgs()
	fullArgs := append(sshArgs, cmdToRunWithSSH)

	cmd := exec.Command("ssh", fullArgs...)
	log.Debugf("RunCommandThroughSSHWithWriters: Full command to run: %v", cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// RunCommandThroughSSH ...
func RunCommandThroughSSH(sshConfigModel config.SSHConfigModel, cmdToRunWithSSH string) error {
	return RunCommandThroughSSHWithWriters(sshConfigModel, cmdToRunWithSSH, os.Stdout, os.Stderr)
}

// GenerateSSHKeypair ...
func GenerateSSHKeypair() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2014)
	if err != nil {
		return []byte{}, []byte{}, err
	}

	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	privateKeyPemBytes := pem.EncodeToMemory(&privateKeyBlock)
	publicKey := privateKey.PublicKey

	pub, err := ssh.NewPublicKey(&publicKey)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	pubBytes := ssh.MarshalAuthorizedKey(pub)

	return privateKeyPemBytes, pubBytes, nil
}
