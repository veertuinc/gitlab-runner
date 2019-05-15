package machine

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/docker"
)

type machineProvider struct {
	name        string
	machine     docker_helpers.Machine
	details     machinesDetails
	lock        sync.RWMutex
	acquireLock sync.Mutex
	// provider stores a real executor that is used to start run the builds
	provider common.ExecutorProvider

	stuckRemoveLock sync.Mutex

	// metrics
	totalActions      *prometheus.CounterVec
	currentStatesDesc *prometheus.Desc
	creationHistogram prometheus.Histogram
}

func (m *machineProvider) machineDetails(name string, acquire bool) *machineDetails {
	m.lock.Lock()
	defer m.lock.Unlock()

	details, ok := m.details[name]
	if !ok {
		details = &machineDetails{
			Name:      name,
			Created:   time.Now(),
			Used:      time.Now(),
			LastSeen:  time.Now(),
			UsedCount: 1, // any machine that we find we mark as already used
			State:     machineStateIdle,
		}
		m.details[name] = details
	}

	if acquire {
		if details.isUsed() {
			return nil
		}
		details.State = machineStateAcquired
	}

	return details
}

func (m *machineProvider) create(config *common.RunnerConfig, state machineState) (details *machineDetails, errCh chan error) {
	name := newMachineName(config)
	details = m.machineDetails(name, true)
	details.State = machineStateCreating
	details.UsedCount = 0
	details.RetryCount = 0
	details.LastSeen = time.Now()
	errCh = make(chan error, 1)

	// Create machine asynchronously
	go func() {
		started := time.Now()
		err := m.machine.Create(config.Machine.MachineDriver, details.Name, config.Machine.MachineOptions...)
		for i := 0; i < 3 && err != nil; i++ {
			details.RetryCount++
			logrus.WithField("name", details.Name).
				WithError(err).
				Warningln("Machine creation failed, trying to provision")
			time.Sleep(provisionRetryInterval)
			err = m.machine.Provision(details.Name)
		}

		if err != nil {
			logrus.WithField("name", details.Name).
				WithField("time", time.Since(started)).
				WithError(err).
				Errorln("Machine creation failed")
			m.remove(details.Name, "Failed to create")
		} else {
			details.State = state
			details.Used = time.Now()
			creationTime := time.Since(started)
			logrus.WithField("time", creationTime).
				WithField("name", details.Name).
				WithField("now", time.Now()).
				WithField("retries", details.RetryCount).
				Infoln("Machine created")
			m.totalActions.WithLabelValues("created").Inc()
			m.creationHistogram.Observe(creationTime.Seconds())
		}
		errCh <- err
	}()
	return
}

func (m *machineProvider) findFreeMachine(skipCache bool, machines ...string) (details *machineDetails) {
	// Enumerate all machines in reverse order, to always take the newest machines first
	for idx := range machines {
		name := machines[len(machines)-idx-1]
		details := m.machineDetails(name, true)
		if details == nil {
			continue
		}

		// Check if node is running
		canConnect := m.machine.CanConnect(name, skipCache)
		if !canConnect {
			m.remove(name, "machine is unavailable")
			continue
		}
		return details
	}

	return nil
}

func (m *machineProvider) useMachine(config *common.RunnerConfig) (details *machineDetails, err error) {
	machines, err := m.loadMachines(config)
	if err != nil {
		return
	}
	details = m.findFreeMachine(true, machines...)
	if details == nil {
		var errCh chan error
		details, errCh = m.create(config, machineStateAcquired)
		err = <-errCh
	}
	return
}

func (m *machineProvider) retryUseMachine(config *common.RunnerConfig) (details *machineDetails, err error) {
	// Try to find a machine
	for i := 0; i < 3; i++ {
		details, err = m.useMachine(config)
		if err == nil {
			break
		}
		time.Sleep(provisionRetryInterval)
	}
	return
}

func (m *machineProvider) removeMachine(details *machineDetails) (err error) {
	if !m.machine.Exist(details.Name) {
		details.logger().
			Warningln("Skipping machine removal, because it doesn't exist")
		return nil
	}

	// This code limits amount of removal of stuck machines to one machine per interval
	if details.isStuckOnRemove() {
		m.stuckRemoveLock.Lock()
		defer m.stuckRemoveLock.Unlock()
	}

	details.logger().
		Warningln("Stopping machine")
	err = m.machine.Stop(details.Name, machineStopCommandTimeout)
	if err != nil {
		details.logger().
			WithError(err).
			Warningln("Error while stopping machine")
	}

	details.logger().
		Warningln("Removing machine")
	err = m.machine.Remove(details.Name)
	if err != nil {
		details.RetryCount++
		time.Sleep(removeRetryInterval)
		return err
	}

	return nil
}

func (m *machineProvider) finalizeRemoval(details *machineDetails) {
	for {
		err := m.removeMachine(details)
		if err == nil {
			break
		}
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.details, details.Name)

	details.logger().
		WithField("now", time.Now()).
		WithField("retries", details.RetryCount).
		Infoln("Machine removed")

	m.totalActions.WithLabelValues("removed").Inc()
}

func (m *machineProvider) remove(machineName string, reason ...interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	details, _ := m.details[machineName]
	if details == nil {
		return errors.New("Machine not found")
	}

	details.Reason = fmt.Sprint(reason...)
	details.State = machineStateRemoving
	details.RetryCount = 0

	details.logger().
		WithField("now", time.Now()).
		Warningln("Requesting machine removal")

	details.Used = time.Now()
	details.writeDebugInformation()

	go m.finalizeRemoval(details)
	return nil
}

func (m *machineProvider) updateMachine(config *common.RunnerConfig, data *machinesData, details *machineDetails) error {
	if details.State != machineStateIdle {
		return nil
	}

	if config.Machine.MaxBuilds > 0 && details.UsedCount >= config.Machine.MaxBuilds {
		// Limit number of builds
		return errors.New("Too many builds")
	}

	if data.Total() >= config.Limit && config.Limit > 0 {
		// Limit maximum number of machines
		return errors.New("Too many machines")
	}

	if time.Since(details.Used) > time.Second*time.Duration(config.Machine.GetIdleTime()) {
		if data.Idle >= config.Machine.GetIdleCount() {
			// Remove machine that are way over the idle time
			return errors.New("Too many idle machines")
		}
	}
	return nil
}

func (m *machineProvider) updateMachines(machines []string, config *common.RunnerConfig) (data machinesData, validMachines []string) {
	data.Runner = config.ShortDescription()
	validMachines = make([]string, 0, len(machines))

	for _, name := range machines {
		details := m.machineDetails(name, false)
		details.LastSeen = time.Now()

		err := m.updateMachine(config, &data, details)
		if err == nil {
			validMachines = append(validMachines, name)
		} else {
			m.remove(details.Name, err)
		}

		data.Add(details)
	}
	return
}

func (m *machineProvider) createMachines(config *common.RunnerConfig, data *machinesData) {
	// Create a new machines and mark them as Idle
	for {
		if data.Available() >= config.Machine.GetIdleCount() {
			// Limit maximum number of idle machines
			break
		}
		if data.Total() >= config.Limit && config.Limit > 0 {
			// Limit maximum number of machines
			break
		}
		m.create(config, machineStateIdle)
		data.Creating++
	}
}

func (m *machineProvider) loadMachines(config *common.RunnerConfig) (machines []string, err error) {
	machines, err = m.machine.List()
	if err != nil {
		return nil, err
	}

	machines = filterMachineList(machines, machineFilter(config))
	return
}

func (m *machineProvider) Acquire(config *common.RunnerConfig) (data common.ExecutorData, err error) {
	if config.Machine == nil || config.Machine.MachineName == "" {
		err = fmt.Errorf("Missing Machine options")
		return
	}

	// Lock updating machines, because two Acquires can be run at the same time
	m.acquireLock.Lock()
	defer m.acquireLock.Unlock()

	machines, err := m.loadMachines(config)
	if err != nil {
		return
	}

	// Update a list of currently configured machines
	machinesData, validMachines := m.updateMachines(machines, config)

	// Pre-create machines
	m.createMachines(config, &machinesData)

	logrus.WithFields(machinesData.Fields()).
		WithField("runner", config.ShortDescription()).
		WithField("minIdleCount", config.Machine.GetIdleCount()).
		WithField("maxMachines", config.Limit).
		WithField("time", time.Now()).
		Debugln("Docker Machine Details")
	machinesData.writeDebugInformation()

	// Try to find a free machine
	details := m.findFreeMachine(false, validMachines...)
	if details != nil {
		data = details
		return
	}

	// If we have a free machines we can process a build
	if config.Machine.GetIdleCount() != 0 && machinesData.Idle == 0 {
		err = errors.New("No free machines that can process builds")
	}
	return
}

func (m *machineProvider) Use(config *common.RunnerConfig, data common.ExecutorData) (newConfig common.RunnerConfig, newData common.ExecutorData, err error) {
	// Find a new machine
	details, _ := data.(*machineDetails)
	if details == nil || !details.canBeUsed() || !m.machine.CanConnect(details.Name, true) {
		details, err = m.retryUseMachine(config)
		if err != nil {
			return
		}

		// Return details only if this is a new instance
		newData = details
	}

	// Get machine credentials
	dc, err := m.machine.Credentials(details.Name)
	if err != nil {
		if newData != nil {
			m.Release(config, newData)
		}
		newData = nil
		return
	}

	// Create shallow copy of config and store in it docker credentials
	newConfig = *config
	newConfig.Docker = &common.DockerConfig{}
	if config.Docker != nil {
		*newConfig.Docker = *config.Docker
	}
	newConfig.Docker.DockerCredentials = dc

	// Mark machine as used
	details.State = machineStateUsed
	details.Used = time.Now()
	details.UsedCount++
	m.totalActions.WithLabelValues("used").Inc()
	return
}

func (m *machineProvider) Release(config *common.RunnerConfig, data common.ExecutorData) {
	// Release machine
	details, ok := data.(*machineDetails)
	if ok {
		// Mark last used time when is Used
		if details.State == machineStateUsed {
			details.Used = time.Now()
		}

		// Remove machine if we already used it
		if config != nil && config.Machine != nil &&
			config.Machine.MaxBuilds > 0 && details.UsedCount >= config.Machine.MaxBuilds {
			err := m.remove(details.Name, "Too many builds")
			if err == nil {
				return
			}
		}
		details.State = machineStateIdle
	}
}

func (m *machineProvider) CanCreate() bool {
	return m.provider.CanCreate()
}

func (m *machineProvider) GetFeatures(features *common.FeaturesInfo) error {
	return m.provider.GetFeatures(features)
}

func (m *machineProvider) GetDefaultShell() string {
	return m.provider.GetDefaultShell()
}

func (m *machineProvider) Create() common.Executor {
	return &machineExecutor{
		provider: m,
	}
}

func newMachineProvider(name, executor string) *machineProvider {
	provider := common.GetExecutor(executor)
	if provider == nil {
		logrus.Panicln("Missing", executor)
	}

	return &machineProvider{
		name:     name,
		details:  make(machinesDetails),
		machine:  docker_helpers.NewMachineCommand(),
		provider: provider,
		totalActions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitlab_runner_autoscaling_actions_total",
				Help: "The total number of actions executed by the provider.",
				ConstLabels: prometheus.Labels{
					"executor": name,
				},
			},
			[]string{"action"},
		),
		currentStatesDesc: prometheus.NewDesc(
			"gitlab_runner_autoscaling_machine_states",
			"The current number of machines per state in this provider.",
			[]string{"state"},
			prometheus.Labels{
				"executor": name,
			},
		),
		creationHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gitlab_runner_autoscaling_machine_creation_duration_seconds",
				Help:    "Histogram of machine creation time.",
				Buckets: prometheus.ExponentialBuckets(30, 1.25, 10),
				ConstLabels: prometheus.Labels{
					"executor": name,
				},
			},
		),
	}
}
