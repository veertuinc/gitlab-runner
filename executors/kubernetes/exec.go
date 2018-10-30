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

This file was modified by James Munnelly (https://gitlab.com/u/munnerz)
*/

package kubernetes

import (
	"fmt"
	"io"
	"net/url"

	"github.com/sirupsen/logrus"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// RemoteExecutor defines the interface accepted by the Exec command - provided for test stubbing
type RemoteExecutor interface {
	Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error
}

// DefaultRemoteExecutor is the standard implementation of remote command execution
type DefaultRemoteExecutor struct{}

func (*DefaultRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}

// ExecOptions declare the arguments accepted by the Exec command
type ExecOptions struct {
	Namespace     string
	PodName       string
	ContainerName string
	Stdin         bool
	Command       []string

	In  io.Reader
	Out io.Writer
	Err io.Writer

	Executor RemoteExecutor
	Client   *kubernetes.Clientset
	Config   *restclient.Config
}

// Run executes a validated remote execution against a pod.
func (p *ExecOptions) Run() error {
	pod, err := p.Client.CoreV1().Pods(p.Namespace).Get(p.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pod.Status.Phase != api.PodRunning {
		return fmt.Errorf("Pod '%s' (on namespace '%s') is not running and cannot execute commands; current phase is '%s'",
			p.PodName, p.Namespace, pod.Status.Phase)
	}

	containerName := p.ContainerName
	if len(containerName) == 0 {
		logrus.Infof("defaulting container name to '%s'", pod.Spec.Containers[0].Name)
		containerName = pod.Spec.Containers[0].Name
	}

	// TODO: refactor with terminal helpers from the edit utility once that is merged
	var stdin io.Reader
	if p.Stdin {
		stdin = p.In
	}

	// TODO: consider abstracting into a client invocation or client helper
	req := p.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", containerName)
	req.VersionedParams(&api.PodExecOptions{
		Container: containerName,
		Command:   p.Command,
		Stdin:     stdin != nil,
		Stdout:    p.Out != nil,
		Stderr:    p.Err != nil,
	}, scheme.ParameterCodec)

	return p.Executor.Execute("POST", req.URL(), p.Config, stdin, p.Out, p.Err, false)
}

func init() {
	runtime.ErrorHandlers = append(runtime.ErrorHandlers, func(err error) {
		logrus.WithError(err).Error("K8S stream error")
	})

	runtime.PanicHandlers = append(runtime.PanicHandlers, func(r interface{}) {
		logrus.Errorf("K8S stream panic: %v", r)
	})
}
