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
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

const JsonContentType = "application/json"

const vmResourcePath = "/api/v1/vm"
const vmRegistryResourcePath = "/api/v1/registry/vm"

type AnkaClient struct {
	controllerAddress     string
	rootCaPath            *string
	certPath              *string
	keyPath               *string
	skipTLSVerification   bool
	controllerHTTPHeaders *string
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

func (ankaClient *AnkaClient) GetNode(nodeID string) (error, *GetNodeResponse) {
	response := GetNodeResponse{}
	nodePath := "/api/v1/node" + "?id=" + nodeID
	err := ankaClient.doRequest("GET", nodePath, nil, &response)
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

func (ankaClient *AnkaClient) GetGroups(nodeGroup string) (error, *GroupResponse) {
	response := GroupResponse{}
	err := ankaClient.doRequest("GET", "/api/v1/group", nil, &response)
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
	logrus.Debugf("urlString: %v\n", urlString)

	timeout := time.Duration(5 * time.Second)

	caCertPool, _ := x509.SystemCertPool()
	if caCertPool == nil {
		caCertPool = x509.NewCertPool()
	}
	if ankaClient.rootCaPath != nil {
		caCert, err := ioutil.ReadFile(*ankaClient.rootCaPath)
		if err != nil {
			return err
		}
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return fmt.Errorf("Could not add %v to Root Certificates", caCert)
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: ankaClient.skipTLSVerification,
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			IdleConnTimeout: 2 * time.Second,
			TLSClientConfig: tlsConfig,
		},
	}

	if ankaClient.certPath != nil {
		if ankaClient.keyPath != nil {
			certs, err := tls.LoadX509KeyPair(*ankaClient.certPath, *ankaClient.keyPath)
			if err != nil {
				return err
			}
			tlsConfig.Certificates = []tls.Certificate{certs}
		} else {
			return errors.New("incomplete key pair... ensure both the cert and key are included")
		}
	}

	// fmt.Printf("tlsConfig: %+v \n", tlsConfig)

	var req *http.Request
	if buffer != nil {
		req, err = http.NewRequest(method, urlString, buffer)
	} else {
		req, err = http.NewRequest(method, urlString, nil)
	}
	if err != nil {
		return err
	}

	if ankaClient.controllerHTTPHeaders != nil {
		var HTTPHeaders map[string]interface{}
		err := json.Unmarshal([]byte(*ankaClient.controllerHTTPHeaders), &HTTPHeaders)
		if err != nil {
			return errors.New("problem with controller http headers")
		}
		for k, v := range HTTPHeaders {
			switch v := v.(type) {
			case string:
				if strings.EqualFold(k, "HOST") { // Can't set HOST in header: https://stackoverflow.com/a/41034588
					req.Host = v
				} else {
					req.Header.Set(k, v)
				}
			default:
				return errors.New("problem with controller http headers (bad JSON; keep it one dimension)")
			}
		}
	}

	req.Header.Set("Content-Type", JsonContentType)

	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		logrus.Errorf("%v\n", err)
	}
	logrus.Debugf("REQUEST TO CONTROLLER: \n %v\n", string(requestDump))

	req.Header.Set("Content-Type", JsonContentType)

	retryLimit := 6
	for tries := 0; tries <= retryLimit; tries++ {
		var response *http.Response
		if tries > 0 {
			time.Sleep(time.Duration(10*tries) * time.Second)
		}
		logrus.Debugf("doRequest retries: %v\n", tries)
		response, err = client.Do(req)
		if err != nil {
			logrus.Debugf("client.Do(req) %v\n", err)
			err = fmt.Errorf("client.Do(req) %v", err)
			continue
		}
		if response == nil || response.Body == nil || response.Status == "" { // If the controller connection fails or is overwhelmed, it will return null or empty values. We need to handle this so the job doesn't fail and orphan VMs.
			if tries == retryLimit {
				err = errors.New("unable to connect to controller... please check its status and cleanup any zombied/orphaned VMs on your Anka Nodes")
				break
			} else {
				logrus.Errorf("something caused the controller to return nil... retrying until we get a valid response..")
				continue
			}
		}
		if err != nil {
			logrus.Errorf("%v\n", err)
			break
		}

		s, _ := json.MarshalIndent(responseBody, "", "\t")

		err = json.NewDecoder(response.Body).Decode(responseBody)
		if response.StatusCode != http.StatusOK {
			logrus.Debugf("json Decode: %v\nResponse: %v\n", err, string(s))
			break
		}
		break
	}

	s, _ := json.MarshalIndent(responseBody, "", "\t")
	logrus.Debugf("RESPONSE FROM CONTROLLER: %v\n", string(s))

	if err != nil {
		return fmt.Errorf("decoding response from controller: %v\nresponse: %v", err, string(s))
	}

	return nil
}

func MakeNewAnkaClient(ankaConfig *common.AnkaConfig) *AnkaClient {
	return &AnkaClient{
		controllerAddress:     ankaConfig.ControllerAddress,
		rootCaPath:            ankaConfig.RootCaPath,
		certPath:              ankaConfig.CertPath,
		controllerHTTPHeaders: ankaConfig.ControllerHTTPHeaders,
		keyPath:               ankaConfig.KeyPath,
		skipTLSVerification:   ankaConfig.SkipTLSVerification,
	}
}
