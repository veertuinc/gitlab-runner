package anka

import (
	"errors"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors/anka/ankaCloudClient"
)

// AnkaConnector is a helper for connecting gitlab runner to anka
type AnkaConnector struct {
	client           *ankaCloudClient.AnkaClient
	netTimeToWait    time.Duration
	startingTimeWait time.Duration
	sshPort          int
}

func (connector *AnkaConnector) StartInstance(ankaConfig *common.AnkaConfig, done *bool, options common.ExecutorPrepareOptions) (connectInfo *AnkaVmConnectInfo, funcErr error) {

	startVmRequest := ankaCloudClient.StartVMRequest{
		VmID:                   ankaConfig.TemplateUUID,
		Tag:                    ankaConfig.Tag,
		NodeID:                 ankaConfig.NodeID,
		Priority:               ankaConfig.Priority,
		GroupId:                ankaConfig.NodeGroup,
		ControllerExternalID:   ankaConfig.ControllerExternalID,
		ControllerInstanceName: ankaConfig.ControllerInstanceName,
	}

	if ankaConfig.MountHostDir != "" {
		startVmRequest.StartupScript = ankaConfig.StartupScript
		startVmRequest.ScriptMonitoring = true
	}

	fmt.Printf("%+v", ankaConfig)
	fmt.Printf("%+v", startVmRequest)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in StartInstance", r)
			funcErr = fmt.Errorf("%v", r)
			if connectInfo.InstanceId != "" {
				connector.TerminateInstance(connectInfo.InstanceId)
			}
		}
	}()

	if ankaConfig.NodeGroup != nil {
		err, groupsResponse := connector.client.GetGroups(*ankaConfig.NodeGroup)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(groupsResponse.Body); i++ { // Check each group for the GroupId the user set. If it matches the name, return the Id instead for StartVM
			if *groupsResponse.Body[i].Id == *ankaConfig.NodeGroup || *groupsResponse.Body[i].Name == *ankaConfig.NodeGroup {
				startVmRequest.GroupId = groupsResponse.Body[i].Id
			}
		}
		if startVmRequest.GroupId == nil {
			return nil, errors.New("The node group ID or name you provided cannot be found")
		}
	}

	// Add name too

	// options.Build.JobInfo.Stage
	if ankaConfig.ControllerInstanceName == "" {
		startVmRequest.ControllerInstanceName = "Anka Gitlab Runner Name: " + fmt.Sprint(options.Build.Runner.Name)
	}

	if ankaConfig.ControllerExternalID == "" {
		startVmRequest.ControllerExternalID = options.Build.JobURL()
	}

	err, createResponse := connector.client.StartVm(&startVmRequest)
	if err != nil {
		return nil, err
	}

	if createResponse.Status != "OK" {
		return nil, errors.New(createResponse.Message)
	}
	if len(createResponse.Body) != 1 {
		return nil, errors.New("No vm id returned from controller") // should never happen
	}

	instanceId := createResponse.Body[0]
	connectInfo = &AnkaVmConnectInfo{
		InstanceId: instanceId,
	}

	// Start the VM and wait for it to pull, etc
	now := time.Now()
	waitForStartUntil := now.Add(connector.startingTimeWait)
	vm, err := connector.waitForVMToStart(instanceId, waitForStartUntil, done)
	if err != nil {
		connector.client.TerminateVm(instanceId)
		return nil, err
	}
	connectInfo.Name = vm.VMInfo.Name
	connectInfo.UUID = vm.VMInfo.Id

	// Get Node Name
	err, node := connector.client.GetNode(vm.VMInfo.NodeId)
	if err != nil {
		connector.client.TerminateVm(instanceId)
		return nil, err
	}
	connectInfo.NodeName = node.Body[0].NodeName
	connectInfo.NodeIP = node.Body[0].IPAddress

	// Wait for the VM to get Networking
	now = time.Now()
	waitForNetworkUntil := now.Add(connector.netTimeToWait)
	vm, err = connector.waitForVMToHaveNetwork(instanceId, waitForNetworkUntil, done)
	if err != nil {
		connector.client.TerminateVm(instanceId)
		return nil, err
	}
	sshPort := connector.getSSHPort(vm)
	if sshPort < 0 {
		connector.client.TerminateVm(instanceId)
		return nil, errors.New("No ssh port forwarding configured on vm")
	}
	connectInfo.Port = sshPort

	sshHost := connector.getSSHHost(vm)
	if sshHost == "" {
		connector.client.TerminateVm(instanceId)
		return nil, errors.New("Unable to determine SSH Host!")
	}
	connectInfo.Host = sshHost
	return connectInfo, funcErr

}

func (connector *AnkaConnector) getVM(instanceId string) (vmStatus *ankaCloudClient.VMStatus, funcErr error) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in getVM", r)
			funcErr = fmt.Errorf("%v", r)
			if instanceId != "" {
				connector.TerminateInstance(instanceId)
			}
		}
	}()

	err, showResponse := connector.client.GetVm(instanceId)
	if err != nil {
		return nil, err
	}
	if showResponse.Status != "OK" {
		return nil, errors.New(showResponse.Message)
	}
	if showResponse.Body.State == "Terminated" || showResponse.Body.State == "Error" { // Handle terminated VMs in the Controller UI
		if showResponse.Body.StartupScript.ReturnCode > 0 {
			return nil, fmt.Errorf("Instance State changed to %v due to failed Startup Script\n%s", showResponse.Body.State, showResponse.Body.StartupScript.Stderr)
		}
		return nil, fmt.Errorf("Instance State changed to %v", showResponse.Body.State)
	}
	return &showResponse.Body, funcErr
}

func (connector *AnkaConnector) waitForVMToStart(instanceId string, timeOut time.Time, done *bool) (vmStatus *ankaCloudClient.VMStatus, funcErr error) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in waitForVMToStart", r)
			funcErr = fmt.Errorf("%v", r)
			if instanceId != "" {
				connector.TerminateInstance(instanceId)
			}
		}
	}()

	for {
		if done != nil { // Handle canceled jobs
			if *done {
				return nil, errors.New("Context.Done() channel received content")
			}
		}
		vm, err := connector.getVM(instanceId)
		if err != nil {
			return nil, err
		}
		loopTime := time.Now()
		switch vm.State {
		case ankaCloudClient.StateStarting:
			fallthrough
		case ankaCloudClient.StateScheduling:
			if loopTime.After(timeOut) {
				return nil, errors.New("VM was unable to start")
			}
			time.Sleep(2 * time.Second)
			break
		case ankaCloudClient.StateStarted:
			return vm, nil
		case ankaCloudClient.StateStopped:
			fallthrough
		case ankaCloudClient.StateStopping:
			fallthrough
		case ankaCloudClient.StateTerminated:
			fallthrough
		case ankaCloudClient.StateTerminating:
			return nil, errors.New("Unexpected VM State(" + string(vm.State) + ")")
		case ankaCloudClient.StateError:
			return nil, errors.New(vm.Message)
		}
	}
	return nil, funcErr
}

func (connector *AnkaConnector) waitForVMToHaveNetwork(instanceId string, timeOut time.Time, done *bool) (vmStatus *ankaCloudClient.VMStatus, funcErr error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in waitForVMToHaveNetwork", r)
			funcErr = fmt.Errorf("%v", r)
			if instanceId != "" {
				connector.TerminateInstance(instanceId)
			}
		}
	}()

	var vm *ankaCloudClient.VMStatus
	for {
		if done != nil { // Handle canceled jobs
			if *done {
				return nil, errors.New("Context.Done() channel received content")
			}
		}
		var err error
		vm, err = connector.getVM(instanceId)
		if err != nil {
			return nil, err
		}
		if connector.checkForNetwork(vm) {
			break
		}
		time.Sleep(2 * time.Second)
		loopTime := time.Now()
		if loopTime.After(timeOut) {
			return nil, errors.New("timeout checking the VM for networking... please review the VM Instance manually to determine why networking didn't start")
		}
	}
	return vm, funcErr
}

func (connector *AnkaConnector) checkForNetwork(vm *ankaCloudClient.VMStatus) bool {
	if vm.State == ankaCloudClient.StateStarted {
		if vm.VMInfo.VmIp != "" {
			return true
		}
	}
	return false
}

func (connector *AnkaConnector) getSSHPort(vm *ankaCloudClient.VMStatus) int {
	if vm.VMInfo.PortForwardingRules != nil {
		for _, portForwardingRule := range *vm.VMInfo.PortForwardingRules {
			if portForwardingRule.VmPort == connector.sshPort {
				return portForwardingRule.NodePort
			}
		}
	}
	return -1
}

func (connector *AnkaConnector) getSSHHost(vm *ankaCloudClient.VMStatus) string {
	return vm.VMInfo.HostIp
}

func (connector *AnkaConnector) TerminateInstance(instanceId string) error {
	err, response := connector.client.TerminateVm(instanceId)
	if err != nil {
		return err
	}
	if response.Status != "OK" {
		return errors.New("could not terminate vm")
	}
	return nil
}

func MakeNewAnkaCloudConnector(ankaConfig *common.AnkaConfig) *AnkaConnector {
	client := ankaCloudClient.MakeNewAnkaClient(ankaConfig)
	return &AnkaConnector{
		client:           client,
		netTimeToWait:    5 * time.Minute,
		startingTimeWait: 90 * time.Minute,
		sshPort:          22,
	}
}

type AnkaVmConnectInfo struct {
	Name       string
	UUID       string
	InstanceId string
	Host       string
	Port       int
	NodeName   string
	NodeIP     string
}
