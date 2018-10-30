package anka

import (
	"strconv"
	"errors"
	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/ssh"

)


type executor struct {
	executors.AbstractExecutor
	sshClient                       ssh.Client 
	vmConnectInfo 		            *AnkaVmConnectInfo
	connector						*AnkaConnector

}
func (s *executor) Prepare(options common.ExecutorPrepareOptions) error {

	s.Debugln("Prepare Anka executor")
	err := s.AbstractExecutor.Prepare(options)
	if err != nil {
		return err
	}

	if s.Config.SSH == nil {
		s.Errorln("No SSH config")
		return errors.New("Missing SSH config")
	}

	if s.Config.Anka == nil {
		s.Errorln("No Anka config")
		return errors.New("Missing Anka configuration")
	}

	if s.Config.Anka.ControllerAddress == "" {
		s.Errorln("No Anka Controller config")

		return errors.New("No Anka Cloud controller configured")
	}

	if s.Config.Anka.ImageId == "" {
		s.Errorln("No Anka image id config")
		return errors.New("Missing image_id from configuration")
	}

	s.connector = MakeNewAnkaCloudConnector(s.Config.Anka.ControllerAddress)
	s.Println("Starting Anka VM from image ", s.Config.Anka.ImageId)
	if s.Config.Anka.Tag != nil {
		s.Println("Tag ", s.Config.Anka.Tag)
	}
	connectInfo, err := s.connector.StartInstance(s.Config.Anka)
	if err != nil {
		return err
	}
	s.Debugln("Connect info: ", connectInfo)
	s.vmConnectInfo = connectInfo
	err = s.verifyMachine()
	if err != nil {
		s.Errorln("unable to verify VM!")
		return err
	}
	s.Println("Anka VM ", connectInfo.InstanceId, " is ready for work")
	err = s.startSSHClient()
	if err != nil {
		s.Errorln(err.Error())
		return err
	}
	
	return nil
}

func (s *executor) verifyMachine() error {
	s.startSSHClient()
	defer s.sshClient.Cleanup()
	err := s.sshClient.Run(s.Context, ssh.Command{Command: []string{"exit"}})
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) startSSHClient() error {
	s.sshClient = ssh.Client{
		Config:  ssh.Config{
			Host: s.vmConnectInfo.Host,
			Port: strconv.Itoa(s.vmConnectInfo.Port),
			User: s.Config.SSH.User,
			Password: s.Config.SSH.Password,
			IdentityFile: s.Config.SSH.IdentityFile,
		},
		Stdout:         s.Trace,
		Stderr:         s.Trace,
		ConnectRetries: 30,
	}
	return s.sshClient.Connect()
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	err := s.sshClient.Run(cmd.Context, ssh.Command{
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
	s.sshClient.Cleanup()
	if s.connector != nil && s.vmConnectInfo != nil {
		if s.Trace.IsJobSuccesFull() || !s.Config.Anka.KeepAliveOnError {
			s.Println("Terminating Anka VM ", s.vmConnectInfo.InstanceId)
			s.connector.TerminateInstance(s.vmConnectInfo.InstanceId)
		}
	}
	s.AbstractExecutor.Cleanup()
}



func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "builds",
		SharedBuildsDir:  false,
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

	common.RegisterExecutor("anka", executors.DefaultExecutorProvider{
		Creator:          creator,
		FeaturesUpdater:  featuresUpdater,
		DefaultShellName: options.Shell.Shell,
	})
}