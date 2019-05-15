package stack

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/test"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

var (
	orchestrator = commonOptions{orchestrator: command.OrchestratorSwarm}
)

func TestListErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		serviceListFunc func(options types.ServiceListOptions) ([]swarm.Service, error)
		expectedError   string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "Template parsing error",
		},
		{
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{}, errors.Errorf("error getting services")
			},
			expectedError: "error getting services",
		},
		{
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*Service()}, nil
			},
			expectedError: "cannot get label",
		},
	}

	for _, tc := range testCases {
		cmd := newListCommand(test.NewFakeCli(&fakeClient{
			serviceListFunc: tc.serviceListFunc,
		}), &orchestrator)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestListWithFormat(t *testing.T) {
	cli := test.NewFakeCli(
		&fakeClient{
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{
					*Service(
						ServiceLabels(map[string]string{
							"com.docker.stack.namespace": "service-name-foo",
						}),
					)}, nil
			},
		})
	cmd := newListCommand(cli, &orchestrator)
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-list-with-format.golden")
}

func TestListWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-foo",
					}),
				)}, nil
		},
	})
	cmd := newListCommand(cli, &orchestrator)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-list-without-format.golden")
}

func TestListOrder(t *testing.T) {
	usecases := []struct {
		golden        string
		swarmServices []swarm.Service
	}{
		{
			golden: "stack-list-sort.golden",
			swarmServices: []swarm.Service{
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-foo",
					}),
				),
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-bar",
					}),
				),
			},
		},
		{
			golden: "stack-list-sort-natural.golden",
			swarmServices: []swarm.Service{
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-1-foo",
					}),
				),
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-10-foo",
					}),
				),
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-2-foo",
					}),
				),
			},
		},
	}

	for _, uc := range usecases {
		cli := test.NewFakeCli(&fakeClient{
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return uc.swarmServices, nil
			},
		})
		cmd := newListCommand(cli, &orchestrator)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), uc.golden)
	}
}
