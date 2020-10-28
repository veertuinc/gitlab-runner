package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/ssh"
	"gitlab.com/gitlab-org/gitlab-runner/network"
)

type configTemplate struct {
	*common.Config

	ConfigFile string `long:"config" env:"TEMPLATE_CONFIG_FILE" description:"Path to the configuration template file"`
}

func (c *configTemplate) Enabled() bool {
	return c.ConfigFile != ""
}

func (c *configTemplate) MergeTo(config *common.RunnerConfig) error {
	err := c.loadConfigTemplate()
	if err != nil {
		return errors.Wrap(err, "couldn't load configuration template file")
	}

	if len(c.Runners) != 1 {
		return errors.New("configuration template must contain exactly one [[runners]] entry")
	}

	err = mergo.Merge(config, c.Runners[0])
	if err != nil {
		return errors.Wrap(err, "error while merging configuration with configuration template")
	}

	return nil
}

func (c *configTemplate) loadConfigTemplate() error {
	config := common.NewConfig()

	err := config.LoadConfig(c.ConfigFile)
	if err != nil {
		return err
	}

	c.Config = config

	return nil
}

//nolint:lll
type RegisterCommand struct {
	context    *cli.Context
	network    common.Network
	reader     *bufio.Reader
	registered bool

	configOptions

	ConfigTemplate configTemplate `namespace:"template"`

	TagList           string `long:"tag-list" env:"RUNNER_TAG_LIST" description:"Tag list"`
	NonInteractive    bool   `short:"n" long:"non-interactive" env:"REGISTER_NON_INTERACTIVE" description:"Run registration unattended"`
	LeaveRunner       bool   `long:"leave-runner" env:"REGISTER_LEAVE_RUNNER" description:"Don't remove runner if registration fails"`
	RegistrationToken string `short:"r" long:"registration-token" env:"REGISTRATION_TOKEN" description:"Runner's registration token"`
	RunUntagged       bool   `long:"run-untagged" env:"REGISTER_RUN_UNTAGGED" description:"Register to run untagged builds; defaults to 'true' when 'tag-list' is empty"`
	Locked            bool   `long:"locked" env:"REGISTER_LOCKED" description:"Lock Runner for current project, defaults to 'true'"`
	AccessLevel       string `long:"access-level" env:"REGISTER_ACCESS_LEVEL" description:"Set access_level of the runner to not_protected or ref_protected; defaults to not_protected"`
	MaximumTimeout    int    `long:"maximum-timeout" env:"REGISTER_MAXIMUM_TIMEOUT" description:"What is the maximum timeout (in seconds) that will be set for job when using this Runner"`
	Paused            bool   `long:"paused" env:"REGISTER_PAUSED" description:"Set Runner to be paused, defaults to 'false'"`
	common.RunnerConfig
}

type AccessLevel string

const (
	NotProtected AccessLevel = "not_protected"
	RefProtected AccessLevel = "ref_protected"
)

const (
	defaultDockerWindowCacheDir = "c:\\cache"
)

func (s *RegisterCommand) askOnce(prompt string, result *string, allowEmpty bool) bool {
	println(prompt)
	if *result != "" {
		print("["+*result, "]: ")
	}

	if s.reader == nil {
		s.reader = bufio.NewReader(os.Stdin)
	}

	data, _, err := s.reader.ReadLine()
	if err != nil {
		panic(err)
	}
	newResult := string(data)
	newResult = strings.TrimSpace(newResult)

	if newResult != "" {
		*result = newResult
		return true
	}

	if allowEmpty || *result != "" {
		return true
	}
	return false
}

func (s *RegisterCommand) ask(key, prompt string, allowEmptyOptional ...bool) string {
	allowEmpty := len(allowEmptyOptional) > 0 && allowEmptyOptional[0]

	result := s.context.String(key)
	result = strings.TrimSpace(result)

	if s.NonInteractive || prompt == "" {
		if result == "" && !allowEmpty {
			logrus.Panicln("The", key, "needs to be entered")
		}
		return result
	}

	for {
		if s.askOnce(prompt, &result, allowEmpty) {
			break
		}
	}

	return result
}

func (s *RegisterCommand) askExecutor() {
	for {
		names := common.GetExecutorNames()
		executors := strings.Join(names, ", ")
		s.Executor = s.ask("executor", "Please enter the executor: "+executors+":", true)
		if common.GetExecutorProvider(s.Executor) != nil {
			return
		}

		message := "Invalid executor specified:"
		if s.NonInteractive {
			logrus.Panicln(message, s.Executor, "(Available:", executors, ")")
		} else {
			logrus.Panicln(message, s.Executor, "(Available: ", executors, ")")
		}
	}
}

func (s *RegisterCommand) askAnka() {
	httpCheck := regexp.MustCompile(`^(http|https)://`)
	s.Anka.ControllerAddress = s.ask("anka-controller-address", "Please enter the Anka Cloud Controller address (http[s]://<address>)")
	if httpCheck.MatchString(s.Anka.ControllerAddress) == false {
		logrus.Panicln("you must use http:// or https://")
	}
	s.Anka.TemplateUUID = s.ask("anka-template-uuid", "Please enter the Anka Template UUID you wish to use for this runner")
	tag := s.ask("anka-tag", "Please enter the Tag name for the Template (leave empty for latest)", true)
	if tag == "" {
		s.Anka.Tag = nil
	} else {
		s.Anka.Tag = &tag
	}

	nodeGroup := s.ask("anka-node-group", "Please enter the Group ID or name you want this runner jobs to be limited to (Enterprise only feature) (leave empty if any node can handle the runner jobs)", true)
	if nodeGroup == "" {
		s.Anka.NodeGroup = nil
	} else {
		s.Anka.NodeGroup = &nodeGroup
	}

	if !s.NonInteractive {
		fmt.Printf("%s%s%s\n", helpers.ANSI_BOLD_YELLOW, "Certificate paths cannot contain a tilde (example: '~/cert.pem')", helpers.ANSI_RESET)
	}
	tildeCheck := regexp.MustCompile("^~/")
	rootCaPath := s.ask("anka-root-ca-path", "[Certificate Authentication] Specify the location of your Controller's Root CA (optional)", true)
	if rootCaPath == "" {
		s.Anka.RootCaPath = nil
	} else {
		s.Anka.RootCaPath = &rootCaPath
	}
	if tildeCheck.MatchString(rootCaPath) == true {
		logrus.Panicln("paths cannot contain tilde (~)")
	}
	certPath := s.ask("anka-cert-path", "[Certificate Authentication] Specify the location of your GitLab Certificate (optional)", true)
	if certPath == "" {
		s.Anka.CertPath = nil
	} else {
		s.Anka.CertPath = &certPath
	}
	if tildeCheck.MatchString(certPath) == true {
		logrus.Panicln("paths cannot contain tilde (~)")
	}
	keyPath := s.ask("anka-key-path", "[Certificate Authentication] Specify the location of your GitLab Certificate Key (optional)", true)
	if keyPath == "" {
		s.Anka.KeyPath = nil
	} else {
		s.Anka.KeyPath = &keyPath
	}
	if tildeCheck.MatchString(keyPath) == true {
		logrus.Panicln("paths cannot contain tilde (~)")
	}
	var err error
	tlsBool := false
	tlsVerification := s.ask("anka-skip-tls-verification", "[Certificate Authentication] Skip TLS Verification? (optional)", true)
	if tlsVerification == "true" || tlsVerification == "false" {
		tlsBool, err = strconv.ParseBool(tlsVerification)
		if err != nil {
			logrus.Panicln(err)
		}
	} else {
		logrus.Panicln("you must provide a boolean (true or false)")
	}
	s.Anka.SkipTLSVerification = tlsBool
}

func (s *RegisterCommand) askSSHServer() {
	s.SSH.Host = s.ask("ssh-host", "Please enter the SSH server address (e.g. my.server.com):")
	s.SSH.Port = s.ask("ssh-port", "Please enter the SSH server port (e.g. 22):", true)
}

func (s *RegisterCommand) askAnkaSSHLogin() {
	s.SSH.User = s.ask("ssh-user", "Please enter the SSH user for your Anka VM (e.g. anka):")
	s.SSH.Password = s.ask("ssh-password", "Please enter the SSH password (e.g. admin):", true)
	tildeCheck := regexp.MustCompile("^~/")
	s.SSH.IdentityFile = s.ask("ssh-identity-file", "Please enter path to SSH identity file (e.g. /home/user/.ssh/id_rsa):", true)
	if tildeCheck.MatchString(s.SSH.IdentityFile) == true {
		logrus.Panicln("paths cannot contain tilde (~)")
	}
}

func (s *RegisterCommand) addRunner(runner *common.RunnerConfig) {
	s.config.Runners = append(s.config.Runners, runner)
}

func (s *RegisterCommand) askRunner() {
	s.URL = s.ask("url", "Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com/):")

	if s.Token != "" {
		logrus.Infoln("Token specified trying to verify runner...")
		logrus.Warningln("If you want to register use the '-r' instead of '-t'.")
		if !s.network.VerifyRunner(s.RunnerCredentials) {
			logrus.Panicln("Failed to verify this runner. Perhaps you are having network problems")
		}
		return
	}

	// we store registration token as token, since we pass that to RunnerCredentials
	s.Token = s.ask("registration-token", "Please enter the gitlab-ci token for this runner:")
	s.Name = s.ask("name", "Please enter the gitlab-ci description for this runner:")
	s.TagList = s.ask("tag-list", "Please enter the gitlab-ci tags for this runner (comma separated):", true)

	if s.TagList == "" {
		s.RunUntagged = true
	}

	parameters := common.RegisterRunnerParameters{
		Description:    s.Name,
		Tags:           s.TagList,
		Locked:         s.Locked,
		AccessLevel:    s.AccessLevel,
		RunUntagged:    s.RunUntagged,
		MaximumTimeout: s.MaximumTimeout,
		Active:         !s.Paused,
	}

	result := s.network.RegisterRunner(s.RunnerCredentials, parameters)
	if result == nil {
		logrus.Panicln("Failed to register this runner. Perhaps you are having network problems")
	}

	s.Token = result.Token
	s.registered = true
}

//nolint:funlen
func (s *RegisterCommand) askExecutorOptions() {

	ssh := s.SSH
	anka := s.Anka

	s.SSH = nil
	s.Referees = nil

	executorFns := map[string]func(){
		"anka": func() {
			s.SSH = ssh
			s.Anka = anka
			s.askAnka()
			s.askAnkaSSHLogin()
		},
	}

	executorFn, ok := executorFns[s.Executor]
	if ok {
		executorFn()
	}
}

func (s *RegisterCommand) Execute(context *cli.Context) {
	userModeWarning(true)

	s.context = context
	err := s.loadConfig()
	if err != nil {
		logrus.Panicln(err)
	}

	validAccessLevels := []AccessLevel{NotProtected, RefProtected}
	if !accessLevelValid(validAccessLevels, AccessLevel(s.AccessLevel)) {
		logrus.Panicln("Given access-level is not valid. " +
			"Please refer to gitlab-runner register -h for the correct options.")
	}

	s.askRunner()

	if !s.LeaveRunner {
		defer s.unregisterRunner()()
	}

	if s.config.Concurrent < s.Limit {
		logrus.Warningf(
			"Specified limit (%d) larger then current concurrent limit (%d). "+
				"Concurrent limit will not be enlarged.",
			s.Limit,
			s.config.Concurrent,
		)
	}

	s.askExecutor()
	s.askExecutorOptions()

	s.mergeTemplate()

	s.addRunner(&s.RunnerConfig)
	err = s.saveConfig()
	if err != nil {
		logrus.Panicln(err)
	}

	logrus.Println("Updated: ", s.ConfigFile)
	logrus.Printf("Feel free to start %v, but if it's running already the config should be automatically reloaded!", common.NAME)

}

func (s *RegisterCommand) unregisterRunner() func() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	go func() {
		signal := <-signals
		s.network.UnregisterRunner(s.RunnerCredentials)
		logrus.Fatalf("RECEIVED SIGNAL: %v", signal)
	}()

	return func() {
		// De-register runner on panic
		if r := recover(); r != nil {
			if s.registered {
				s.network.UnregisterRunner(s.RunnerCredentials)
			}

			// pass panic to next defer
			panic(r)
		}
	}
}

// TODO: Remove in 13.0 https://gitlab.com/gitlab-org/gitlab-runner/issues/6404
//
// transformDockerServices will take the value from `DockerServices`
// and convert the value of each entry into a `common.DockerService` definition.
//
// This is to keep backward compatibility when the user passes
// `--docker-services alpine:3.11 --docker-services ruby:3.10` we parse this
// correctly and create the service definition.
// func (s *RegisterCommand) transformDockerServices(services []string) {
// 	for _, service := range services {
// 		s.Docker.Services = append(
// 			s.Docker.Services,
// 			&common.DockerService{
// 				Service: common.Service{Name: service},
// 			},
// 		)
// 	}
// }

func (s *RegisterCommand) mergeTemplate() {
	if !s.ConfigTemplate.Enabled() {
		return
	}

	logrus.Infof("Merging configuration from template file %q", s.ConfigTemplate.ConfigFile)

	err := s.ConfigTemplate.MergeTo(&s.RunnerConfig)
	if err != nil {
		logrus.WithError(err).Fatal("Could not handle configuration merging from template file")
	}
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func newRegisterCommand() *RegisterCommand {
	return &RegisterCommand{
		RunnerConfig: common.RunnerConfig{
			Name: getHostname(),
			RunnerSettings: common.RunnerSettings{
				Anka: &common.AnkaConfig{},
				SSH:  &ssh.Config{},
			},
		},
		Locked:  true,
		Paused:  false,
		network: network.NewGitLabClient(),
	}
}

func accessLevelValid(levels []AccessLevel, givenLevel AccessLevel) bool {
	if givenLevel == "" {
		return true
	}

	for _, level := range levels {
		if givenLevel == level {
			return true
		}
	}

	return false
}

func init() {
	common.RegisterCommand2("register", "register a new runner", newRegisterCommand())
}
