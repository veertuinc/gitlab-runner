package parallels

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/ssh"

	prl "gitlab.com/gitlab-org/gitlab-runner/helpers/parallels"
)

type executor struct {
	executors.AbstractExecutor
	cmd             *exec.Cmd
	vmName          string
	sshCommand      ssh.Client
	provisioned     bool
	ipAddress       string
	machineVerified bool
}

func (s *executor) waitForIPAddress(vmName string, seconds int) (string, error) {
	var lastError error

	if s.ipAddress != "" {
		return s.ipAddress, nil
	}

	s.Debugln("Looking for MAC address...")
	macAddr, err := prl.Mac(vmName)
	if err != nil {
		return "", err
	}

	s.Debugln("Requesting IP address...")
	for i := 0; i < seconds; i++ {
		ipAddr, err := prl.IPAddress(macAddr)
		if err == nil {
			s.Debugln("IP address found", ipAddr, "...")
			s.ipAddress = ipAddr
			return ipAddr, nil
		}
		lastError = err
		time.Sleep(time.Second)
	}
	return "", lastError
}

func (s *executor) verifyMachine(vmName string) error {
	if s.machineVerified {
		return nil
	}

	ipAddr, err := s.waitForIPAddress(vmName, 120)
	if err != nil {
		return err
	}

	// Create SSH command
	sshCommand := ssh.Client{
		Config:         *s.Config.SSH,
		Stdout:         s.Trace,
		Stderr:         s.Trace,
		ConnectRetries: 30,
	}
	sshCommand.Host = ipAddr

	s.Debugln("Connecting to SSH...")
	err = sshCommand.Connect()
	if err != nil {
		return err
	}
	defer sshCommand.Cleanup()
	err = sshCommand.Run(s.Context, ssh.Command{Command: []string{"exit"}})
	if err != nil {
		return err
	}
	s.machineVerified = true
	return nil
}

func (s *executor) restoreFromSnapshot() error {
	s.Debugln("Requesting default snapshot for VM...")
	snapshot, err := prl.GetDefaultSnapshot(s.vmName)
	if err != nil {
		return err
	}

	s.Debugln("Reverting VM to snapshot", snapshot, "...")
	err = prl.RevertToSnapshot(s.vmName, snapshot)
	if err != nil {
		return err
	}

	return nil
}

func (s *executor) createVM() error {
	baseImage := s.Config.Parallels.BaseName
	if baseImage == "" {
		return errors.New("Missing Image setting from Parallels config")
	}

	templateName := s.Config.Parallels.TemplateName
	if templateName == "" {
		templateName = baseImage + "-template"
	}

	// remove invalid template (removed?)
	templateStatus, _ := prl.Status(templateName)
	if templateStatus == prl.Invalid {
		prl.Unregister(templateName)
	}

	if !prl.Exist(templateName) {
		s.Debugln("Creating template from VM", baseImage, "...")
		err := prl.CreateTemplate(baseImage, templateName)
		if err != nil {
			return err
		}
	}

	s.Debugln("Creating runner from VM template...")
	err := prl.CreateOsVM(s.vmName, templateName)
	if err != nil {
		return err
	}

	s.Debugln("Bootstraping VM...")
	err = prl.Start(s.vmName)
	if err != nil {
		return err
	}

	// TODO: integration tests do fail on this due
	// Unable to open new session in this virtual machine.
	// Make sure the latest version of Parallels Tools is installed in this virtual machine and it has finished bootingg
	s.Debugln("Waiting for VM to start...")
	err = prl.TryExec(s.vmName, 120, "exit", "0")
	if err != nil {
		return err
	}

	s.Debugln("Waiting for VM to become responsive...")
	err = s.verifyMachine(s.vmName)
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) updateGuestTime() error {
	s.Debugln("Updating VM date...")
	timeServer := s.Config.Parallels.TimeServer
	if timeServer == "" {
		timeServer = "time.apple.com"
	}

	// Check either ntpdate command exists or not before trying to execute it
	// Starting from Mojave ntpdate was removed
	_, err := prl.Exec(s.vmName, "which", "ntpdate")
	if err != nil {
		// Fallback to sntp
		return prl.TryExec(s.vmName, 20, "sudo", "sntp", "-sS", timeServer)
	}

	return prl.TryExec(s.vmName, 20, "sudo", "ntpdate", "-u", timeServer)
}

func (s *executor) Prepare(options common.ExecutorPrepareOptions) error {
	err := s.AbstractExecutor.Prepare(options)
	if err != nil {
		return err
	}

	if s.BuildShell.PassFile {
		return errors.New("Parallels doesn't support shells that require script file")
	}

	if s.Config.SSH == nil {
		return errors.New("Missing SSH configuration")
	}

	if s.Config.Parallels == nil {
		return errors.New("Missing Parallels configuration")
	}

	if s.Config.Parallels.BaseName == "" {
		return errors.New("Missing BaseName setting from Parallels config")
	}

	version, err := prl.Version()
	if err != nil {
		return err
	}

	s.Println("Using Parallels", version, "executor...")

	// remove invalid VM (removed?)
	vmStatus, _ := prl.Status(s.vmName)
	if vmStatus == prl.Invalid {
		prl.Unregister(s.vmName)
	}

	if s.Config.Parallels.DisableSnapshots {
		s.vmName = s.Config.Parallels.BaseName + "-" + s.Build.ProjectUniqueName()
		if prl.Exist(s.vmName) {
			s.Debugln("Deleting old VM...")
			prl.Kill(s.vmName)
			prl.Delete(s.vmName)
			prl.Unregister(s.vmName)
		}
	} else {
		s.vmName = fmt.Sprintf("%s-runner-%s-concurrent-%d",
			s.Config.Parallels.BaseName,
			s.Build.Runner.ShortDescription(),
			s.Build.RunnerID)
	}

	if prl.Exist(s.vmName) {
		s.Println("Restoring VM from snapshot...")
		err := s.restoreFromSnapshot()
		if err != nil {
			s.Println("Previous VM failed. Deleting, because", err)
			prl.Kill(s.vmName)
			prl.Delete(s.vmName)
			prl.Unregister(s.vmName)
		}
	}

	if !prl.Exist(s.vmName) {
		s.Println("Creating new VM...")
		err := s.createVM()
		if err != nil {
			return err
		}

		if !s.Config.Parallels.DisableSnapshots {
			s.Println("Creating default snapshot...")
			err = prl.CreateSnapshot(s.vmName, "Started")
			if err != nil {
				return err
			}
		}
	}

	s.Debugln("Checking VM status...")
	status, err := prl.Status(s.vmName)
	if err != nil {
		return err
	}

	// Start VM if stopped
	if status == prl.Stopped || status == prl.Suspended {
		s.Println("Starting VM...")
		err := prl.Start(s.vmName)
		if err != nil {
			return err
		}
	}

	if status != prl.Running {
		s.Debugln("Waiting for VM to run...")
		err = prl.WaitForStatus(s.vmName, prl.Running, 60)
		if err != nil {
			return err
		}
	}

	s.Println("Waiting VM to become responsive...")
	err = s.verifyMachine(s.vmName)
	if err != nil {
		return err
	}

	s.provisioned = true

	// TODO: integration tests do fail on this due
	// Unable to open new session in this virtual machine.
	// Make sure the latest version of Parallels Tools is installed in this virtual machine and it has finished booting
	err = s.updateGuestTime()
	if err != nil {
		s.Println("Could not sync with timeserver!")
		return err
	}

	ipAddr, err := s.waitForIPAddress(s.vmName, 60)
	if err != nil {
		return err
	}

	s.Debugln("Starting SSH command...")
	s.sshCommand = ssh.Client{
		Config: *s.Config.SSH,
		Stdout: s.Trace,
		Stderr: s.Trace,
	}
	s.sshCommand.Host = ipAddr

	s.Debugln("Connecting to SSH server...")
	err = s.sshCommand.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	err := s.sshCommand.Run(cmd.Context, ssh.Command{
		Environment: s.BuildShell.Environment,
		Command:     s.BuildShell.GetCommandWithArguments(),
		Stdin:       cmd.Script,
	})
	if _, ok := err.(*ssh.ExitError); ok {
		err = &common.BuildError{Inner: err}
	}
	return err
}

func (s *executor) Cleanup() {
	s.sshCommand.Cleanup()

	if s.vmName != "" {
		prl.Kill(s.vmName)

		if s.Config.Parallels.DisableSnapshots || !s.provisioned {
			prl.Delete(s.vmName)
		}
	}

	s.AbstractExecutor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultCustomBuildsDirEnabled: false,
		DefaultBuildsDir:              "builds",
		SharedBuildsDir:               false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.LoginShell,
			RunnerCommand: "gitlab-runner",
		},
		ShowHostname: true,
	}

	creator := func() common.Executor {
		return &executor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
	}

	common.RegisterExecutor("parallels", executors.DefaultExecutorProvider{
		Creator:          creator,
		FeaturesUpdater:  featuresUpdater,
		DefaultShellName: options.Shell.Shell,
	})
}
