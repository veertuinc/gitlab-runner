package ankaCloudClient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

const JsonContentType = "application/json"

const vmResourcePath = "/api/v1/vm"
const vmRegistryResourcePath = "/api/v1/registry/vm"

type AnkaClient struct {
	controllerAddress string
	rootCaPath        *string
	certPath          *string
	keyPath           *string
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
	if err != nil {
		return err
	}
	urlObj = urlObj.ResolveReference(relativePath)
	if err != nil {
		return err
	}
	urlString := urlObj.String()
	fmt.Println(urlString)

	client := &http.Client{}

	caCertPool := x509.NewCertPool()

	if ankaClient.rootCaPath != nil {
		caCert, err := ioutil.ReadFile(*ankaClient.rootCaPath)
		if err != nil {
			return err
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	if ankaClient.certPath != nil {
		if ankaClient.keyPath != nil {
			cert, err := tls.LoadX509KeyPair(*ankaClient.certPath, *ankaClient.keyPath)
			if err != nil {
				return err
			}
			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:      caCertPool,
						Certificates: []tls.Certificate{cert},
					},
				},
			}

		} else {
			return errors.New("incomplete key pair... ensure both the cert and key are included")
		}
	}

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

	err = json.NewDecoder(response.Body).Decode(responseBody)
	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("%v %+v", response.Status, responseBody)) // TODO: Only output message
	}

	fmt.Println("Response ", responseBody)
	if err != nil {
		return err
	}
	return nil

}

func MakeNewAnkaClient(ankaConfig *common.AnkaConfig) *AnkaClient {
	return &AnkaClient{
		controllerAddress: ankaConfig.ControllerAddress,
		rootCaPath:        ankaConfig.RootCaPath,
		certPath:          ankaConfig.CertPath,
		keyPath:           ankaConfig.KeyPath,
	}
}
