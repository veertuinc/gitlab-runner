package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/network"
)

type UnregisterCommand struct {
	configOptions
	common.RunnerCredentials
	network    common.Network
	Name       string `toml:"name" json:"name" short:"n" long:"name" description:"Name of the runner you wish to unregister"`
	AllRunners bool   `toml:"all_runners" json:"all-runners" long:"all-runners" description:"Unregister all runners"`
}

func (c *UnregisterCommand) unregisterAllRunners() (runners []*common.RunnerConfig) {
	logrus.Warningln("Unregistering all runners")
	for _, r := range c.config.Runners {
		if !c.network.UnregisterRunner(r.RunnerCredentials) {
			logrus.Errorln("Failed to unregister runner", r.Name)
			//If unregister fails, leave the runner in the config
			runners = append(runners, r)
		}
	}
	return
}

func (c *UnregisterCommand) unregisterSingleRunner() (runners []*common.RunnerConfig) {
	if len(c.Name) > 0 { // Unregister when given a name
		runnerConfig, err := c.RunnerByName(c.Name)
		if err != nil {
			logrus.Fatalln(err)
		}
		c.RunnerCredentials = runnerConfig.RunnerCredentials
	}

	// Unregister given Token and URL of the runner
	if !c.network.UnregisterRunner(c.RunnerCredentials) {
		logrus.Fatalln("Failed to unregister runner", c.Name)
	}

	for _, otherRunner := range c.config.Runners {
		if otherRunner.RunnerCredentials == c.RunnerCredentials {
			continue
		}
		runners = append(runners, otherRunner)
	}
	return
}

func (c *UnregisterCommand) Execute(context *cli.Context) {
	userModeWarning(false)

	err := c.loadConfig()
	if err != nil {
		logrus.Fatalln(err)
		return
	}

	var runners []*common.RunnerConfig
	if c.AllRunners {
		runners = c.unregisterAllRunners()
	} else {
		runners = c.unregisterSingleRunner()
	}

	// check if anything changed
	if len(c.config.Runners) == len(runners) {
		return
	}

	c.config.Runners = runners

	// save config file
	err = c.saveConfig()
	if err != nil {
		logrus.Fatalln("Failed to update", c.ConfigFile, err)
	}
	logrus.Println("Updated", c.ConfigFile)
}

func init() {
	common.RegisterCommand2("unregister", "unregister specific runner", &UnregisterCommand{
		network: network.NewGitLabClient(),
	})
}
