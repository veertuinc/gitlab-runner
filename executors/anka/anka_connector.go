package anka

import (
	"gitlab.com/gitlab-org/gitlab-runner/common"
	"errors"
	"github.com/asafg6/anka-client"
	"time"
)

type AnkaConnector struct {
	client *ankaCloudClient.AnkaClient
	timeToWait 	int
	sshPort     int
}

func (connector *AnkaConnector) StartInstance(ankaConfig *common.AnkaConfig) (*AnkaVmConnectInfo, error) {
	startVmRequset := ankaCloudClient.StartVMRequest {
		VmID: ankaConfig.ImageId,
		Tag: ankaConfig.Tag,
		NodeID: ankaConfig.NodeID,
		Priority: ankaConfig.Priority,
		GroupId: ankaConfig.GroupId,		
	}
	err, createResponse := connector.client.StartVm(&startVmRequset)
	if err != nil {
		return nil, err
	}
	if createResponse.Status != "OK" {
		return nil, errors.New(createResponse.Message)
	}
	if len(createResponse.Body) != 1 {
		return nil, errors.New("No vm id returened from controller") // should never happen
	}
	instanceId := createResponse.Body[0]
	timeOut := time.Now()
	timeOut = timeOut.Add(time.Duration(connector.timeToWait) * time.Second)
	var vm ankaCloudClient.VMStatus
	for {
		err, showResponse := connector.client.GetVm(instanceId)
		if err != nil {
			connector.client.TerminateVm(instanceId)
			return nil, err
		}
		if showResponse.Status != "OK" {
			return nil, errors.New(showResponse.Message)
		}
		vm = showResponse.Body
		if connector.checkForNetwork(vm) {
			break
		}
		time.Sleep(2 * time.Second)
		loopTime := time.Now()
		if loopTime.After(timeOut) {
			connector.client.TerminateVm(instanceId)
			return nil, errors.New("VM could not get network")
		}
	}
	sshPort := connector.getSSHPort(vm)
	if sshPort < 0 {
		connector.client.TerminateVm(instanceId)
		return nil, errors.New("No ssh port forwarding configured on vm")
	}
	sshHost := connector.getSSHHost(vm)
	return &AnkaVmConnectInfo{
		InstanceId: instanceId,
		Port: sshPort,
		Host: sshHost,
	}, nil

}

func (connector *AnkaConnector) checkForNetwork(vm ankaCloudClient.VMStatus) bool {
	if vm.State == ankaCloudClient.StateStarted {
		if vm.VMInfo.VmIp != "" {
			return true
		}
	}
	return false
}

func (connector *AnkaConnector) getSSHPort(vm ankaCloudClient.VMStatus) int {
	if vm.VMInfo.PortForwardingRules != nil {
		for _, portForwardingRule := range *vm.VMInfo.PortForwardingRules {
			if portForwardingRule.VmPort == connector.sshPort {
				return portForwardingRule.NodePort
			}
		}
	}
	return -1
}


func (connector *AnkaConnector) getSSHHost(vm ankaCloudClient.VMStatus) string {
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


func MakeNewAnkaCloudConnector(ankaControllerAddress string) *AnkaConnector {
	client := ankaCloudClient.MakeNewAnkaClient(ankaControllerAddress)
	return &AnkaConnector{
		client: client,
		timeToWait: 120,
		sshPort: 22,
	}
}

type AnkaVmConnectInfo struct {
	InstanceId string
	Host	string
	Port	int
}