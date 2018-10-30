package commands

import (
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gitlab.com/ayufan/golang-cli-helpers"
	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/gitlab_ci_yaml_parser"

	// Force to load all executors, executes init() on them
	_ "gitlab.com/gitlab-org/gitlab-runner/executors/docker"
	_ "gitlab.com/gitlab-org/gitlab-runner/executors/parallels"
	_ "gitlab.com/gitlab-org/gitlab-runner/executors/shell"
	_ "gitlab.com/gitlab-org/gitlab-runner/executors/ssh"
	_ "gitlab.com/gitlab-org/gitlab-runner/executors/virtualbox"
)

type ExecCommand struct {
	common.RunnerSettings
	Job     string
	Timeout int `long:"timeout" description:"Job execution timeout (in seconds)"`
}

func (c *ExecCommand) runCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	result, err := cmd.Output()
	return string(result), err
}

func (c *ExecCommand) createBuild(repoURL string, abortSignal chan os.Signal) (build *common.Build, err error) {
	// Check if we have uncommitted changes
	_, err = c.runCommand("git", "diff", "--quiet", "HEAD")
	if err != nil {
		logrus.Warningln("You most probably have uncommitted changes.")
		logrus.Warningln("These changes will not be tested.")
	}

	// Parse Git settings
	sha, err := c.runCommand("git", "rev-parse", "HEAD")
	if err != nil {
		return
	}

	beforeSha, err := c.runCommand("git", "rev-parse", "HEAD~1")
	if err != nil {
		beforeSha = "0000000000000000000000000000000000000000"
	}

	refName, err := c.runCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return
	}

	jobResponse := common.JobResponse{
		ID:            1,
		Token:         "",
		AllowGitFetch: false,
		JobInfo: common.JobInfo{
			Name:        "",
			Stage:       "",
			ProjectID:   1,
			ProjectName: "",
		},
		GitInfo: common.GitInfo{
			RepoURL:   repoURL,
			Ref:       strings.TrimSpace(refName),
			Sha:       strings.TrimSpace(sha),
			BeforeSha: strings.TrimSpace(beforeSha),
		},
		RunnerInfo: common.RunnerInfo{
			Timeout: c.getTimeout(),
		},
	}

	runner := &common.RunnerConfig{
		RunnerSettings: c.RunnerSettings,
	}

	build = common.NewBuild(jobResponse, runner, abortSignal, nil)

	return
}

func (c *ExecCommand) getTimeout() int {
	if c.Timeout > 0 {
		return c.Timeout
	}

	return common.DefaultExecTimeout
}

func (c *ExecCommand) Execute(context *cli.Context) {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalln(err)
	}

	switch len(context.Args()) {
	case 1:
		c.Job = context.Args().Get(0)
	default:
		cli.ShowSubcommandHelp(context)
		os.Exit(1)
		return
	}

	c.Executor = context.Command.Name

	abortSignal := make(chan os.Signal)
	doneSignal := make(chan int, 1)

	go waitForInterrupts(nil, abortSignal, doneSignal, nil)

	// Add self-volume to docker
	if c.RunnerSettings.Docker == nil {
		c.RunnerSettings.Docker = &common.DockerConfig{}
	}
	c.RunnerSettings.Docker.Volumes = append(c.RunnerSettings.Docker.Volumes, wd+":"+wd+":ro")

	// Create build
	build, err := c.createBuild(wd, abortSignal)
	if err != nil {
		logrus.Fatalln(err)
	}

	parser := gitlab_ci_yaml_parser.NewGitLabCiYamlParser(c.Job)
	err = parser.ParseYaml(&build.JobResponse)
	if err != nil {
		logrus.Fatalln(err)
	}

	err = build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	if err != nil {
		logrus.Fatalln(err)
	}
}

func init() {
	cmd := &ExecCommand{}

	flags := clihelpers.GetFlagsFromStruct(cmd)
	cliCmd := cli.Command{
		Name:  "exec",
		Usage: "execute a build locally",
	}

	for _, executor := range common.GetExecutors() {
		subCmd := cli.Command{
			Name:   executor,
			Usage:  "use " + executor + " executor",
			Action: cmd.Execute,
			Flags:  flags,
		}
		cliCmd.Subcommands = append(cliCmd.Subcommands, subCmd)
	}

	common.RegisterCommand(cliCmd)
}
