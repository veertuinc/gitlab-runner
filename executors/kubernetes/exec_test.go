/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
)

type fakeRemoteExecutor struct {
	method  string
	url     *url.URL
	execErr error
}

func (f *fakeRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	f.method = method
	f.url = url
	return f.execErr
}

func TestExec(t *testing.T) {
	version, codec := testVersionAndCodec()
	tests := []struct {
		name, version, podPath, execPath string
		pod                              *api.Pod
		tty, execErr                     bool
	}{
		{
			name:     "pod exec",
			version:  version,
			podPath:  "/api/" + version + "/namespaces/test/pods/foo",
			execPath: "/api/" + version + "/namespaces/test/pods/foo/exec",
			pod:      execPod(),
		},
		{
			name:     "pod exec with tty",
			version:  version,
			podPath:  "/api/" + version + "/namespaces/test/pods/foo",
			execPath: "/api/" + version + "/namespaces/test/pods/foo/exec",
			pod:      execPod(),
			tty:      true,
		},
		{
			name:     "pod exec error",
			version:  version,
			podPath:  "/api/" + version + "/namespaces/test/pods/foo",
			execPath: "/api/" + version + "/namespaces/test/pods/foo/exec",
			pod:      execPod(),
			execErr:  true,
		},
	}

	for _, test := range tests {
		// Create a fake kubeClient
		fakeClient := fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch p, m := req.URL.Path, req.Method; {
			case p == test.podPath && m == "GET":
				body := objBody(codec, test.pod)
				return &http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{
					"Content-Type": []string{"application/json"},
				}}, nil
			default:
				// Ensures no GET is performed when deleting by name
				t.Errorf("%s: unexpected request: %s %#v\n%#v", test.name, req.Method, req.URL, req)
				return nil, fmt.Errorf("unexpected request")
			}
		})
		c := testKubernetesClient(version, fakeClient)

		ex := &fakeRemoteExecutor{}
		if test.execErr {
			ex.execErr = fmt.Errorf("exec error")
		}

		bufOut := bytes.NewBuffer([]byte{})
		bufErr := bytes.NewBuffer([]byte{})
		bufIn := bytes.NewBuffer([]byte{})

		params := &ExecOptions{
			PodName:       "foo",
			ContainerName: "bar",
			Namespace:     "test",
			Command:       []string{"command"},
			In:            bufIn,
			Out:           bufOut,
			Err:           bufErr,
			Stdin:         true,
			Executor:      ex,
			Client:        c,
		}
		err := params.Run()
		if test.execErr && err != ex.execErr {
			t.Errorf("%s: Unexpected exec error: %v", test.name, err)
			continue
		}
		if !test.execErr && err != nil {
			t.Errorf("%s: Unexpected error: %v", test.name, err)
			continue
		}
		if test.execErr {
			continue
		}
		if ex.url.Path != test.execPath {
			t.Errorf("%s: Did not get expected path for exec request", test.name)
			continue
		}
		if ex.method != "POST" {
			t.Errorf("%s: Did not get method for exec request: %s", test.name, ex.method)
		}
	}
}

func execPod() *api.Pod {
	return &api.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "test", ResourceVersion: "10"},
		Spec: api.PodSpec{
			RestartPolicy: api.RestartPolicyAlways,
			DNSPolicy:     api.DNSClusterFirst,
			Containers: []api.Container{
				{
					Name: "bar",
				},
			},
		},
		Status: api.PodStatus{
			Phase: api.PodRunning,
		},
	}
}
