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
	"net/http/httputil"
	"net/url"
	"time"

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

	timeout := time.Duration(5 * time.Second)

	client := &http.Client{
		Timeout: timeout,
	}

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
				Timeout: timeout,
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

	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("REQUEST TO CONTROLLER: \n %v\n", string(requestDump))

	req.Header.Set("Content-Type", JsonContentType)

	retryLimit := 6
	for tries := 0; tries <= retryLimit; tries++ {
		fmt.Printf("Retry: %v\n", tries)
		response, err := client.Do(req)
		if response == nil || response.Body == nil || response.Status == "" { // If the controller connection fails or is overwhelmed, it will return null or empty values. We need to handle this so the job doesn't fail and orphan VMs.
			if tries == retryLimit {
				err = errors.New("unable to connect to controller... please check its status and cleanup any zombied/orphaned VMs on your Anka Nodes")
				break
			} else {
				fmt.Printf("something caused the controller to return nill... retrying until we get a valid retry...")
				time.Sleep(time.Duration(10*tries) * time.Second)
				continue
			}
		}
		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}
		err = json.NewDecoder(response.Body).Decode(responseBody)
		if response.StatusCode != http.StatusOK {
			fmt.Printf("%v\n", err)
			break
		}
		break
	}
	if err != nil {
		return err
	}

	s, _ := json.MarshalIndent(responseBody, "", "\t")
	fmt.Printf("RESPONSE FROM CONTROLLER: %v\n", string(s))

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
