package ankaCloudClient

import (
	"errors"
	"fmt"
	"net/url"
	"encoding/json"
	"bytes"
	"net/http"
)

const JsonContentType = "application/json"

const vmResourcePath = "/api/v1/vm"
const vmRegistryResourcePath = "/api/v1/registry/vm"


type AnkaClient struct {
	controllerAddress string
}

func (ankaClient *AnkaClient) GetVms() (error, *ListVmResponse) {
	response := ListVmResponse{}
	err := ankaClient.doRequest("GET", vmResourcePath, nil, &response)
	if err != nil {
		return err, nil
	}
	return nil, &response

}

func (ankaClient *AnkaClient) GetVm(instanceId string) (error, *GetVmResponse) {
	response := GetVmResponse{}
	vmPath := vmResourcePath + "?id=" + instanceId
	err := ankaClient.doRequest("GET", vmPath, nil, &response)
	if err != nil {
		return err, nil
	}
	return nil, &response
}

func (ankaClient *AnkaClient) GetRegistryVms() (error, *RegistryVmResponse) {
	response := RegistryVmResponse{}
	err := ankaClient.doRequest("GET", vmRegistryResourcePath, nil, &response)
	if err != nil {
		return err, nil
	}
	return nil, &response

}

func (ankaClient *AnkaClient) StartVm(startVmRequest *StartVMRequest) (error, *StartVmResponse) {
	response := StartVmResponse{}
	err := ankaClient.doRequest("POST", vmResourcePath, startVmRequest, &response)
	if err != nil {
		return err, nil
	}
	return nil, &response
}

func (ankaClient *AnkaClient) TerminateVm(instanceId string) (error, *StandardResponse) {
	response := StandardResponse{}
	requestBody := TerminateVMRequest{InstanceID: instanceId}
	err := ankaClient.doRequest("DELETE", vmResourcePath, &requestBody, &response)
	if err != nil {
		return err, nil
	}
	return nil, &response
}

func (ankaClient *AnkaClient) doRequest(method string, path string, body interface{}, responseBody interface{}) error {
	var buffer *bytes.Buffer
	var err error
	if body != nil {
		buffer = new(bytes.Buffer)
		err := json.NewEncoder(buffer).Encode(body)
		if err != nil {
			return err
		}
	} else {
		buffer = nil
	}
	relativePath, _ := url.ParseRequestURI(path)
	urlObj, err := url.ParseRequestURI(ankaClient.controllerAddress)
	urlObj = urlObj.ResolveReference(relativePath)
	if err != nil {
		return err
	}
	urlString := urlObj.String()
	fmt.Println(urlString)
	client := &http.Client{}
	var req *http.Request
	if buffer != nil {
		req, err = http.NewRequest(method, urlString, buffer)
	} else {
		req, err = http.NewRequest(method, urlString, nil)
	}
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", JsonContentType)
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("%v %v", response.StatusCode, response.Body))
	}

	err = json.NewDecoder(response.Body).Decode(responseBody)
	fmt.Println("Response ", responseBody)
	if err != nil {
		return err
	}
	return nil

}

func MakeNewAnkaClient(ankaCloudControllerUrl string) *AnkaClient{
	return &AnkaClient{
		controllerAddress: ankaCloudControllerUrl,
	}
}