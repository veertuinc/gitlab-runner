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

func (connector *AnkaConnector) StartInstance(ankaConfig *common.AnkaConfig) (connectInfo *AnkaVmConnectInfo, funcErr error) {

	startVmRequest := ankaCloudClient.StartVMRequest{
		VmID:     ankaConfig.TemplateUUID,
		Tag:      ankaConfig.Tag,
		NodeID:   ankaConfig.NodeID,
		Priority: ankaConfig.Priority,
		GroupId:  ankaConfig.GroupId,
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in StartInstance", r)
			funcErr = errors.New("enexpected error")
			if connectInfo.InstanceId != "" {
				connector.TerminateInstance(connectInfo.InstanceId)
			}
		}
	}()

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

	now := time.Now()
	waitForStartUntil := now.Add(connector.startingTimeWait)

	err, vm := connector.waitForVMToStart(instanceId, waitForStartUntil)
	if err != nil {
		connector.client.TerminateVm(instanceId)
		return nil, err
	}

	now = time.Now()
	waitForNetworUntil := now.Add(connector.netTimeToWait)
	err, vm = connector.waitForVMToHaveNetwork(instanceId, waitForNetworUntil)
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

func (connector *AnkaConnector) getVM(instanceId string) (error, *ankaCloudClient.VMStatus) {
	err, showResponse := connector.client.GetVm(instanceId)
	if err != nil {
		return err, nil
	}
	if showResponse.Status != "OK" {
		return errors.New(showResponse.Message), nil
	}
	return nil, &showResponse.Body
}

func (connector *AnkaConnector) waitForVMToStart(instanceId string, timeOut time.Time) (error, *ankaCloudClient.VMStatus) {
	for {
		err, vm := connector.getVM(instanceId)
		if err != nil {
			return err, nil
		}
		loopTime := time.Now()
		switch vm.State {

		case ankaCloudClient.StateStarting:
			fallthrough
		case ankaCloudClient.StateScheduling:
			if loopTime.After(timeOut) {
				return errors.New("VM was unable to start"), nil
			}
			time.Sleep(2 * time.Second)
			break

		case ankaCloudClient.StateStarted:
			return nil, vm

		case ankaCloudClient.StateStopped:
			fallthrough
		case ankaCloudClient.StateStopping:
			fallthrough
		case ankaCloudClient.StateTerminated:
			fallthrough
		case ankaCloudClient.StateTerminating:
			return errors.New("Unexpected VM State " + string(vm.State)), nil

		case ankaCloudClient.StateError:
			return errors.New(vm.Message), nil

		}
	}
	return nil, nil
}

func (connector *AnkaConnector) waitForVMToHaveNetwork(instanceId string, timeOut time.Time) (error, *ankaCloudClient.VMStatus) {

	var vm *ankaCloudClient.VMStatus
	for {
		var err error
		err, vm = connector.getVM(instanceId)
		if err != nil {
			return err, nil
		}
		if connector.checkForNetwork(vm) {
			break
		}
		time.Sleep(2 * time.Second)
		loopTime := time.Now()
		if loopTime.After(timeOut) {
			return errors.New("timeout checking the VM for networking... please review the VM Instance manually to determine why networking didn't start"), nil
		}
	}
	return nil, vm
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
	InstanceId string
	Host       string
	Port       int
}
