package kubernetes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"gitlab.com/gitlab-org/gitlab-terminal"
	"golang.org/x/net/context"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	// Register all available authentication methods
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/dns"
	terminalsession "gitlab.com/gitlab-org/gitlab-runner/session/terminal"
)

var (
	executorOptions = executors.ExecutorOptions{
		DefaultCustomBuildsDirEnabled: true,
		DefaultBuildsDir:              "/builds",
		DefaultCacheDir:               "/cache",
		SharedBuildsDir:               false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.NormalShell,
			RunnerCommand: "/usr/bin/gitlab-runner-helper",
		},
		ShowHostname: true,
	}
)

type kubernetesOptions struct {
	Image    common.Image
	Services common.Services
}

type executor struct {
	executors.AbstractExecutor

	kubeClient  *kubernetes.Clientset
	pod         *api.Pod
	credentials *api.Secret
	options     *kubernetesOptions

	configurationOverwrites *overwrites
	buildLimits             api.ResourceList
	serviceLimits           api.ResourceList
	helperLimits            api.ResourceList
	buildRequests           api.ResourceList
	serviceRequests         api.ResourceList
	helperRequests          api.ResourceList
	pullPolicy              common.KubernetesPullPolicy
}

func (s *executor) setupResources() error {
	var err error

	// Limit
	if s.buildLimits, err = limits(s.Config.Kubernetes.CPULimit, s.Config.Kubernetes.MemoryLimit); err != nil {
		return fmt.Errorf("invalid build limits specified: %s", err.Error())
	}

	if s.serviceLimits, err = limits(s.Config.Kubernetes.ServiceCPULimit, s.Config.Kubernetes.ServiceMemoryLimit); err != nil {
		return fmt.Errorf("invalid service limits specified: %s", err.Error())
	}

	if s.helperLimits, err = limits(s.Config.Kubernetes.HelperCPULimit, s.Config.Kubernetes.HelperMemoryLimit); err != nil {
		return fmt.Errorf("invalid helper limits specified: %s", err.Error())
	}

	// Requests
	if s.buildRequests, err = limits(s.Config.Kubernetes.CPURequest, s.Config.Kubernetes.MemoryRequest); err != nil {
		return fmt.Errorf("invalid build requests specified: %s", err.Error())
	}

	if s.serviceRequests, err = limits(s.Config.Kubernetes.ServiceCPURequest, s.Config.Kubernetes.ServiceMemoryRequest); err != nil {
		return fmt.Errorf("invalid service requests specified: %s", err.Error())
	}

	if s.helperRequests, err = limits(s.Config.Kubernetes.HelperCPURequest, s.Config.Kubernetes.HelperMemoryRequest); err != nil {
		return fmt.Errorf("invalid helper requests specified: %s", err.Error())
	}
	return nil
}

func (s *executor) Prepare(options common.ExecutorPrepareOptions) (err error) {
	if err = s.AbstractExecutor.Prepare(options); err != nil {
		return err
	}

	if s.BuildShell.PassFile {
		return fmt.Errorf("kubernetes doesn't support shells that require script file")
	}

	if err = s.setupResources(); err != nil {
		return err
	}

	if s.pullPolicy, err = s.Config.Kubernetes.PullPolicy.Get(); err != nil {
		return err
	}

	if err = s.prepareOverwrites(options.Build.Variables); err != nil {
		return err
	}

	s.prepareOptions(options.Build)

	if err = s.checkDefaults(); err != nil {
		return err
	}

	if s.kubeClient, err = getKubeClient(options.Config.Kubernetes, s.configurationOverwrites); err != nil {
		return fmt.Errorf("error connecting to Kubernetes: %s", err.Error())
	}

	s.Println("Using Kubernetes executor with image", s.options.Image.Name, "...")

	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	s.Debugln("Starting Kubernetes command...")

	if s.pod == nil {
		err := s.setupCredentials()
		if err != nil {
			return err
		}

		err = s.setupBuildPod()
		if err != nil {
			return err
		}
	}

	containerName := "build"
	containerCommand := s.BuildShell.DockerCommand
	if cmd.Predefined {
		containerName = "helper"
		containerCommand = common.ContainerCommandBuild
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Debugln(fmt.Sprintf(
		"Starting in container %q the command %q with script: %s",
		containerName,
		containerCommand,
		cmd.Script,
	))

	select {
	case err := <-s.runInContainer(ctx, containerName, containerCommand, cmd.Script):
		s.Debugln(fmt.Sprintf("Container %q exited with error: %v", containerName, err))
		if err != nil && strings.Contains(err.Error(), "command terminated with exit code") {
			return &common.BuildError{Inner: err}
		}
		return err

	case <-cmd.Context.Done():
		return fmt.Errorf("build aborted")
	}
}

func (s *executor) Cleanup() {
	if s.pod != nil {
		err := s.kubeClient.CoreV1().Pods(s.pod.Namespace).Delete(s.pod.Name, &metav1.DeleteOptions{})
		if err != nil {
			s.Errorln(fmt.Sprintf("Error cleaning up pod: %s", err.Error()))
		}
	}
	if s.credentials != nil {
		err := s.kubeClient.CoreV1().Secrets(s.configurationOverwrites.namespace).Delete(s.credentials.Name, &metav1.DeleteOptions{})
		if err != nil {
			s.Errorln(fmt.Sprintf("Error cleaning up secrets: %s", err.Error()))
		}
	}
	closeKubeClient(s.kubeClient)
	s.AbstractExecutor.Cleanup()
}

func (s *executor) buildContainer(name, image string, imageDefinition common.Image, requests, limits api.ResourceList, containerCommand ...string) api.Container {
	privileged := false
	if s.Config.Kubernetes != nil {
		privileged = s.Config.Kubernetes.Privileged
	}

	command, args := s.getCommandAndArgs(imageDefinition, containerCommand...)

	return api.Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: api.PullPolicy(s.pullPolicy),
		Command:         command,
		Args:            args,
		Env:             buildVariables(s.Build.GetAllVariables().PublicOrInternal()),
		Resources: api.ResourceRequirements{
			Limits:   limits,
			Requests: requests,
		},
		VolumeMounts: s.getVolumeMounts(),
		SecurityContext: &api.SecurityContext{
			Privileged: &privileged,
		},
		Stdin: true,
	}
}

func (s *executor) getCommandAndArgs(imageDefinition common.Image, command ...string) ([]string, []string) {
	if s.Build.IsFeatureFlagOn("FF_K8S_USE_ENTRYPOINT_OVER_COMMAND") {
		return s.getCommandsAndArgsV2(imageDefinition, command...)
	}

	return s.getCommandsAndArgsV1(imageDefinition, command...)
}

// TODO: Remove in 12.0
func (s *executor) getCommandsAndArgsV1(imageDefinition common.Image, command ...string) ([]string, []string) {
	if len(command) == 0 && len(imageDefinition.Command) > 0 {
		command = imageDefinition.Command
	}

	var args []string
	if len(imageDefinition.Entrypoint) > 0 {
		args = command
		command = imageDefinition.Entrypoint
	}

	return command, args
}

// TODO: Make this the only proper way to setup command and args in 12.0
func (s *executor) getCommandsAndArgsV2(imageDefinition common.Image, command ...string) ([]string, []string) {
	if len(command) == 0 && len(imageDefinition.Entrypoint) > 0 {
		command = imageDefinition.Entrypoint
	}

	var args []string
	if len(imageDefinition.Command) > 0 {
		args = imageDefinition.Command
	}

	return command, args
}

func (s *executor) getVolumeMounts() (mounts []api.VolumeMount) {
	mounts = append(mounts, api.VolumeMount{
		Name:      "repo",
		MountPath: s.Build.RootDir,
	})

	for _, mount := range s.Config.Kubernetes.Volumes.HostPaths {
		mounts = append(mounts, api.VolumeMount{
			Name:      mount.Name,
			MountPath: mount.MountPath,
			ReadOnly:  mount.ReadOnly,
		})
	}

	for _, mount := range s.Config.Kubernetes.Volumes.Secrets {
		mounts = append(mounts, api.VolumeMount{
			Name:      mount.Name,
			MountPath: mount.MountPath,
			ReadOnly:  mount.ReadOnly,
		})
	}

	for _, mount := range s.Config.Kubernetes.Volumes.PVCs {
		mounts = append(mounts, api.VolumeMount{
			Name:      mount.Name,
			MountPath: mount.MountPath,
			ReadOnly:  mount.ReadOnly,
		})
	}

	for _, mount := range s.Config.Kubernetes.Volumes.ConfigMaps {
		mounts = append(mounts, api.VolumeMount{
			Name:      mount.Name,
			MountPath: mount.MountPath,
			ReadOnly:  mount.ReadOnly,
		})
	}

	for _, mount := range s.Config.Kubernetes.Volumes.EmptyDirs {
		mounts = append(mounts, api.VolumeMount{
			Name:      mount.Name,
			MountPath: mount.MountPath,
		})
	}

	return
}

func (s *executor) getVolumes() (volumes []api.Volume) {
	volumes = append(volumes, api.Volume{
		Name: "repo",
		VolumeSource: api.VolumeSource{
			EmptyDir: &api.EmptyDirVolumeSource{},
		},
	})

	for _, volume := range s.Config.Kubernetes.Volumes.HostPaths {
		path := volume.HostPath
		// Make backward compatible with syntax introduced in version 9.3.0
		if path == "" {
			path = volume.MountPath
		}

		volumes = append(volumes, api.Volume{
			Name: volume.Name,
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: path,
				},
			},
		})
	}

	for _, volume := range s.Config.Kubernetes.Volumes.Secrets {
		items := []api.KeyToPath{}
		for key, path := range volume.Items {
			items = append(items, api.KeyToPath{Key: key, Path: path})
		}

		volumes = append(volumes, api.Volume{
			Name: volume.Name,
			VolumeSource: api.VolumeSource{
				Secret: &api.SecretVolumeSource{
					SecretName: volume.Name,
					Items:      items,
				},
			},
		})
	}

	for _, volume := range s.Config.Kubernetes.Volumes.PVCs {
		volumes = append(volumes, api.Volume{
			Name: volume.Name,
			VolumeSource: api.VolumeSource{
				PersistentVolumeClaim: &api.PersistentVolumeClaimVolumeSource{
					ClaimName: volume.Name,
					ReadOnly:  volume.ReadOnly,
				},
			},
		})
	}

	for _, volume := range s.Config.Kubernetes.Volumes.ConfigMaps {
		items := []api.KeyToPath{}
		for key, path := range volume.Items {
			items = append(items, api.KeyToPath{Key: key, Path: path})
		}

		volumes = append(volumes, api.Volume{
			Name: volume.Name,
			VolumeSource: api.VolumeSource{
				ConfigMap: &api.ConfigMapVolumeSource{
					LocalObjectReference: api.LocalObjectReference{
						Name: volume.Name,
					},
					Items: items,
				},
			},
		})
	}

	for _, volume := range s.Config.Kubernetes.Volumes.EmptyDirs {
		volumes = append(volumes, api.Volume{
			Name: volume.Name,
			VolumeSource: api.VolumeSource{
				EmptyDir: &api.EmptyDirVolumeSource{
					Medium: api.StorageMedium(volume.Medium),
				},
			},
		})
	}

	return
}

type dockerConfigEntry struct {
	Username, Password string
}

func (s *executor) projectUniqueName() string {
	return dns.MakeRFC1123Compatible(s.Build.ProjectUniqueName())
}

func (s *executor) setupCredentials() error {
	authConfigs := make(map[string]dockerConfigEntry)

	for _, credentials := range s.Build.Credentials {
		if credentials.Type != "registry" {
			continue
		}

		authConfigs[credentials.URL] = dockerConfigEntry{
			Username: credentials.Username,
			Password: credentials.Password,
		}
	}

	if len(authConfigs) == 0 {
		return nil
	}

	dockerCfgContent, err := json.Marshal(authConfigs)
	if err != nil {
		return err
	}

	secret := api.Secret{}
	secret.GenerateName = s.projectUniqueName()
	secret.Namespace = s.configurationOverwrites.namespace
	secret.Type = api.SecretTypeDockercfg
	secret.Data = map[string][]byte{}
	secret.Data[api.DockerConfigKey] = dockerCfgContent

	s.credentials, err = s.kubeClient.CoreV1().Secrets(s.configurationOverwrites.namespace).Create(&secret)
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) setupBuildPod() error {
	services := make([]api.Container, len(s.options.Services))
	for i, service := range s.options.Services {
		resolvedImage := s.Build.GetAllVariables().ExpandValue(service.Name)
		services[i] = s.buildContainer(fmt.Sprintf("svc-%d", i), resolvedImage, service, s.serviceRequests, s.serviceLimits)
	}

	labels := make(map[string]string)
	for k, v := range s.Build.Runner.Kubernetes.PodLabels {
		labels[k] = s.Build.Variables.ExpandValue(v)
	}

	annotations := make(map[string]string)
	for key, val := range s.configurationOverwrites.podAnnotations {
		annotations[key] = s.Build.Variables.ExpandValue(val)
	}

	var imagePullSecrets []api.LocalObjectReference
	for _, imagePullSecret := range s.Config.Kubernetes.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, api.LocalObjectReference{Name: imagePullSecret})
	}

	if s.credentials != nil {
		imagePullSecrets = append(imagePullSecrets, api.LocalObjectReference{Name: s.credentials.Name})
	}

	buildImage := s.Build.GetAllVariables().ExpandValue(s.options.Image.Name)
	helperImage := common.AppVersion.Variables().ExpandValue(s.Config.Kubernetes.GetHelperImage())

	pod, err := s.kubeClient.CoreV1().Pods(s.configurationOverwrites.namespace).Create(&api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: s.projectUniqueName(),
			Namespace:    s.configurationOverwrites.namespace,
			Labels:       labels,
			Annotations:  annotations,
		},
		Spec: api.PodSpec{
			Volumes:            s.getVolumes(),
			ServiceAccountName: s.configurationOverwrites.serviceAccount,
			RestartPolicy:      api.RestartPolicyNever,
			NodeSelector:       s.Config.Kubernetes.NodeSelector,
			Tolerations:        s.Config.Kubernetes.GetNodeTolerations(),
			Containers: append([]api.Container{
				// TODO use the build and helper template here
				s.buildContainer("build", buildImage, s.options.Image, s.buildRequests, s.buildLimits, s.BuildShell.DockerCommand...),
				s.buildContainer("helper", helperImage, common.Image{}, s.helperRequests, s.helperLimits, s.BuildShell.DockerCommand...),
			}, services...),
			TerminationGracePeriodSeconds: &s.Config.Kubernetes.TerminationGracePeriodSeconds,
			ImagePullSecrets:              imagePullSecrets,
		},
	})

	if err != nil {
		return err
	}

	s.pod = pod

	return nil
}

func (s *executor) runInContainer(ctx context.Context, name string, command []string, script string) <-chan error {
	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		status, err := waitForPodRunning(ctx, s.kubeClient, s.pod, s.Trace, s.Config.Kubernetes)

		if err != nil {
			errc <- err
			return
		}

		if status != api.PodRunning {
			errc <- fmt.Errorf("pod failed to enter running state: %s", status)
			return
		}

		config, err := getKubeClientConfig(s.Config.Kubernetes, s.configurationOverwrites)

		if err != nil {
			errc <- err
			return
		}

		exec := ExecOptions{
			PodName:       s.pod.Name,
			Namespace:     s.pod.Namespace,
			ContainerName: name,
			Command:       command,
			In:            strings.NewReader(script),
			Out:           s.Trace,
			Err:           s.Trace,
			Stdin:         true,
			Config:        config,
			Client:        s.kubeClient,
			Executor:      &DefaultRemoteExecutor{},
		}

		errc <- exec.Run()
	}()

	return errc
}

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
	proxy := terminal.NewWebSocketProxy(1) // one stopper: terminal exit handler

	terminalsession.ProxyTerminal(
		timeoutCh,
		disconnectCh,
		proxy.StopCh,
		func() {
			terminal.ProxyWebSocket(w, r, t.settings, proxy)
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

	if wsURL.Scheme == "https" {
		wsURL.Scheme = "wss"
	} else if wsURL.Scheme == "http" {
		wsURL.Scheme = "ws"
	}

	return wsURL, nil
}

func (s *executor) prepareOverwrites(variables common.JobVariables) error {
	values, err := createOverwrites(s.Config.Kubernetes, variables, s.BuildLogger)
	if err != nil {
		return err
	}

	s.configurationOverwrites = values
	return nil
}

func (s *executor) prepareOptions(job *common.Build) {
	s.options = &kubernetesOptions{}
	s.options.Image = job.Image
	for _, service := range job.Services {
		if service.Name == "" {
			continue
		}
		s.options.Services = append(s.options.Services, service)
	}
}

// checkDefaults Defines the configuration for the Pod on Kubernetes
func (s *executor) checkDefaults() error {
	if s.options.Image.Name == "" {
		if s.Config.Kubernetes.Image == "" {
			return fmt.Errorf("no image specified and no default set in config")
		}

		s.options.Image = common.Image{
			Name: s.Config.Kubernetes.Image,
		}
	}

	if s.configurationOverwrites.namespace == "" {
		s.Warningln("Namespace is empty, therefore assuming 'default'.")
		s.configurationOverwrites.namespace = "default"
	}

	s.Println("Using Kubernetes namespace:", s.configurationOverwrites.namespace)

	return nil
}

func createFn() common.Executor {
	return &executor{
		AbstractExecutor: executors.AbstractExecutor{
			ExecutorOptions: executorOptions,
		},
	}
}

func featuresFn(features *common.FeaturesInfo) {
	features.Variables = true
	features.Image = true
	features.Services = true
	features.Artifacts = true
	features.Cache = true
	features.Session = true
	features.Terminal = true
}

func init() {
	common.RegisterExecutor("kubernetes", executors.DefaultExecutorProvider{
		Creator:          createFn,
		FeaturesUpdater:  featuresFn,
		DefaultShellName: executorOptions.Shell.Shell,
	})
}
