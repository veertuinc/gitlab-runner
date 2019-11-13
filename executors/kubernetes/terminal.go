package kubernetes

import (
	"io/ioutil"
	"net/http"
	"net/url"

	terminal "gitlab.com/gitlab-org/gitlab-terminal"
	api "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"

	"gitlab.com/gitlab-org/gitlab-runner/session/proxy"
	terminalsession "gitlab.com/gitlab-org/gitlab-runner/session/terminal"
)

func (s *executor) Connect() (terminalsession.Conn, error) {
	settings, err := s.getTerminalSettings()
	if err != nil {
		return nil, err
	}

	return terminalConn{settings: settings}, nil
}

type terminalConn struct {
	settings *terminal.TerminalSettings
}

func (t terminalConn) Start(w http.ResponseWriter, r *http.Request, timeoutCh, disconnectCh chan error) {
	wsProxy := terminal.NewWebSocketProxy(1) // one stopper: terminal exit handler

	terminalsession.ProxyTerminal(
		timeoutCh,
		disconnectCh,
		wsProxy.StopCh,
		func() {
			terminal.ProxyWebSocket(w, r, t.settings, wsProxy)
		},
	)
}

func (t terminalConn) Close() error {
	return nil
}

func (s *executor) getTerminalSettings() (*terminal.TerminalSettings, error) {
	config, err := getKubeClientConfig(s.Config.Kubernetes, s.configurationOverwrites)
	if err != nil {
		return nil, err
	}

	wsURL, err := s.getTerminalWebSocketURL(config)
	if err != nil {
		return nil, err
	}

	caCert := ""
	if len(config.CAFile) > 0 {
		buf, err := ioutil.ReadFile(config.CAFile)
		if err != nil {
			return nil, err
		}
		caCert = string(buf)
	}

	term := &terminal.TerminalSettings{
		Subprotocols:   []string{"channel.k8s.io"},
		Url:            wsURL.String(),
		Header:         http.Header{"Authorization": []string{"Bearer " + config.BearerToken}},
		CAPem:          caCert,
		MaxSessionTime: 0,
	}

	return term, nil
}

func (s *executor) getTerminalWebSocketURL(config *restclient.Config) (*url.URL, error) {
	wsURL := s.kubeClient.CoreV1().RESTClient().Post().
		Namespace(s.pod.Namespace).
		Resource("pods").
		Name(s.pod.Name).
		SubResource("exec").
		VersionedParams(&api.PodExecOptions{
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
			Container: "build",
			Command:   []string{"sh", "-c", "bash || sh"},
		}, scheme.ParameterCodec).URL()

	wsURL.Scheme = proxy.WebsocketProtocolFor(wsURL.Scheme)
	return wsURL, nil
}
