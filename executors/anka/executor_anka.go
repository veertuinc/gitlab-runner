package anka

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/ssh"
)

// anka-executor

type executor struct {
	executors.AbstractExecutor
	sshClient     ssh.Client
	vmConnectInfo *AnkaVmConnectInfo
	connector     *AnkaConnector
}

func LogAndUIPrint(s *executor, options common.ExecutorPrepareOptions, input string) {
	options.Config.Log().WithFields(logrus.Fields{
		"job": options.Build.JobResponse.ID,
	}).Println(input)
	s.Println(input)
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

	if s.Config.Anka.TemplateUUID == "" {
		s.Errorln("No Anka image id config")
		return errors.New("Missing template_uuid from configuration")
	}

	ankaTemplateUUIDENV := s.Build.Variables.Get("ANKA_TEMPLATE_UUID")
	if ankaTemplateUUIDENV != "" { // OVERRIDE of default Template
		s.Config.Anka.TemplateUUID = ankaTemplateUUIDENV
	}

	ankaTagNameENV := s.Build.Variables.Get("ANKA_TAG_NAME")
	if ankaTagNameENV != "" { // OVERRIDE of default Tag
		s.Config.Anka.Tag = &ankaTagNameENV
	}

	ankaGroupENV := s.Build.Variables.Get("ANKA_NODE_GROUP")
	if ankaGroupENV != "" {
		s.Config.Anka.NodeGroup = &ankaGroupENV
	}

	s.Println("Opening a connection to the Anka Cloud Controller:", s.Config.Anka.ControllerAddress)
	s.connector = MakeNewAnkaCloudConnector(s.Config.Anka)

	s.Println(fmt.Sprintf("%s%s%s", helpers.ANSI_BOLD_CYAN, "Starting Anka VM using:", helpers.ANSI_RESET))
	s.Println("  - Template UUID:", s.Config.Anka.TemplateUUID)
	if s.Config.Anka.Tag != nil {
		s.Println("  - Tag:", *s.Config.Anka.Tag)
	}
	if s.Config.Anka.NodeGroup != nil {
		s.Println("  - Node Group:", *s.Config.Anka.NodeGroup)
	}

	s.Println("Please be patient...")

	// Handle canceled jobs in the UI
	done := false
	go func() {
		doneChannel := s.Context.Done()
		<-doneChannel
		done = true
	}()

	s.Println(fmt.Sprintf("%s %s/#/instances", "You can check the status of starting your Instance on the Anka Cloud Controller:", s.Config.Anka.ControllerAddress))

	connectInfo, err := s.connector.StartInstance(s.Config.Anka, &done)
	if err != nil {
		return err
	}

	s.vmConnectInfo = connectInfo
	s.Println(fmt.Sprintf("Verifying connectivity to the VM - Host: %v Port: %v Instance UUID: %v ", s.vmConnectInfo.Host, s.vmConnectInfo.Port, s.vmConnectInfo.InstanceId))
	err = s.verifyNode()
	if err != nil {
		LogAndUIPrint(s, options, fmt.Sprint("SSH Error to VM:", err, s.vmConnectInfo))
		return err
	}

	LogAndUIPrint(s, options, fmt.Sprintf("%sAnka VM %s, running on Node %s (%s), is ready for work (%s:%v%s)", helpers.ANSI_BOLD_GREEN, connectInfo.InstanceId, connectInfo.NodeName, connectInfo.NodeIP, connectInfo.Host, connectInfo.Port, helpers.ANSI_RESET))

	err = s.startSSHClient()
	if err != nil {
		LogAndUIPrint(s, options, fmt.Sprint(err.Error()))
		return err
	}

	return nil
}

func (s *executor) verifyNode() error {
	defer s.sshClient.Cleanup()
	err := s.startSSHClient()
	if err != nil {
		return err
	}
	err = s.sshClient.Run(s.Context, ssh.Command{Command: []string{"exit"}})
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) startSSHClient() error {
	s.sshClient = ssh.Client{
		Config: ssh.Config{
			Host:         s.vmConnectInfo.Host,
			Port:         strconv.Itoa(s.vmConnectInfo.Port),
			User:         s.Config.SSH.User,
			Password:     s.Config.SSH.Password,
			IdentityFile: s.Config.SSH.IdentityFile,
		},
		Stdout:         s.Trace,
		Stderr:         s.Trace,
		ConnectRetries: 1,
	}
	err := s.sshClient.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	logrus.Debugf("%+v\n", ssh.Command{
		Environment: s.BuildShell.Environment,
		Command:     s.BuildShell.GetCommandWithArguments(),
		Stdin:       cmd.Script,
	})
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
		if s.Trace.IsJobSuccessful() || !s.Config.Anka.KeepAliveOnError {
			s.Println("Terminating Anka VM ", s.vmConnectInfo.InstanceId)
			s.connector.TerminateInstance(s.vmConnectInfo.InstanceId)
		}
	}
	s.AbstractExecutor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "builds",
		DefaultCacheDir:  "cache",
		SharedBuildsDir:  false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.LoginShell,
			RunnerCommand: "anka-gitlab-runner",
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

	common.RegisterExecutorProvider("anka", executors.DefaultExecutorProvider{
		Creator:          creator,
		FeaturesUpdater:  featuresUpdater,
		DefaultShellName: options.Shell.Shell,
	})
}
