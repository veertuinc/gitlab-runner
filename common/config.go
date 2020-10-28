package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/ssh"
	"gitlab.com/gitlab-org/gitlab-runner/referees"
)

type DockerPullPolicy string
type DockerSysCtls map[string]string

const (
	PullPolicyAlways       = "always"
	PullPolicyNever        = "never"
	PullPolicyIfNotPresent = "if-not-present"
)

// InvalidTimePeriodsError represents that the time period specified is not valid.
type InvalidTimePeriodsError struct {
	periods []string
	cause   error
}

func NewInvalidTimePeriodsError(periods []string, cause error) *InvalidTimePeriodsError {
	return &InvalidTimePeriodsError{periods: periods, cause: cause}
}

func (e *InvalidTimePeriodsError) Error() string {
	return fmt.Sprintf("invalid time periods %v, caused by: %v", e.periods, e.cause)
}

func (e *InvalidTimePeriodsError) Is(err error) bool {
	_, ok := err.(*InvalidTimePeriodsError)

	return ok
}

func (e *InvalidTimePeriodsError) Unwrap() error {
	return e.cause
}

type AnkaConfig struct {
	ControllerAddress     string  `toml:"controller_address" json:"controller_address" long:"controller-address" env:"CONTROLLER_ADDRESS" description:"Anka Cloud Controller address (example: http://anka-controller.mydomain.net[:8090])"`
	TemplateUUID          string  `toml:"template_uuid" json:"template_uuid" long:"template-uuid" env:"TEMPLATE_UUID" description:"Specify the VM Template UUID"`
	Tag                   *string `toml:"tag,omitempty" json:"tag" long:"tag" env:"TAG" description:"Specify the Tag to use"`
	NodeID                *string `toml:"node_id,omitempty" json:"node_id" long:"node-id" env:"NODE_ID" description:"Specify the Node ID to run the job (you can find this in your Controller's Nodes page)"`
	Priority              *int    `toml:"priority,omitzero" json:"priority" long:"priority" env:"PRIORITY" description:"Set the job priority"`
	NodeGroup             *string `toml:"node_group,omitempty" json:"node_group" long:"node-group" env:"NODE_GROUP" description:"Limit jobs to a specific node group (accepts name or ID)"`
	RootCaPath            *string `toml:"root_ca_path,omitempty" json:"root_ca_path" long:"root-ca-path" env:"ROOT_CA_PATH" description:"Specify the path to your Controller's Root CA certificate"`
	CertPath              *string `toml:"cert_path,omitempty" json:"cert_path" long:"cert-path" env:"CERT_PATH" description:"Specify the path to the GitLab Certificate (used for connecting to the Controller) (requires you also specify the key)"`
	KeyPath               *string `toml:"key_path,omitempty" json:"key_path" long:"key-path" env:"KEY_PATH" description:"Specify the path to your GitLab Certificate Key (used for connecting to the Controller)"`
	ControllerHTTPHeaders *string `toml:"controller_http_headers,omitempty" json:"controller_http_headers" long:"controller-http-headers" env:"CONTROLLER_HTTP_HEADERS" description:"In JSON format, specify headers to set for the HTTP requests to the controller (quotes must be escaped) (example: "{ \"HOST\": \"testing123.com\", \"CustomHeaderName\": \"test123\" }")"`
	// Be sure to use *bool or else setting --anka-skip-tls-verification true will ignore anything after it when you're doing register --non-interactive
	SkipTLSVerification bool `toml:"skip_tls_verification,omitzero" json:"skip_tls_verification" long:"skip-tls-verification" env:"SKIP_TLS_VERIFICATION" description:"Skip TLS Verification when connecting to your Controller"`
	KeepAliveOnError    bool `toml:"keep_alive_on_error,omitzero" json:"keep_alive_on_error" long:"keep-alive-on-error" env:"KEEP_ALIVE_ON_ERROR" description:"Keep the VM alive for debugging job failures"`
}

func tryGetTomlValue(data map[string]interface{}, key string) (string, error) {
	value, ok := data[key]
	if !ok {
		return "", nil
	}

	switch v := value.(type) {
	case string:
		return v, nil
	}

	return "", fmt.Errorf("toml: cannot load TOML value of type %T into a Go string", value)
}

type Service struct {
	Name  string `toml:"name" long:"name" description:"The image path for the service"`
	Alias string `toml:"alias,omitempty" long:"alias" description:"The alias of the service"`
}

func (s *Service) ToImageDefinition() Image {
	return Image{
		Name:  s.Name,
		Alias: s.Alias,
	}
}

//nolint:lll
type RunnerCredentials struct {
	URL         string `toml:"url" json:"url" short:"u" long:"url" env:"CI_SERVER_URL" required:"true" description:"Runner URL"`
	Token       string `toml:"token" json:"token" short:"t" long:"token" env:"CI_SERVER_TOKEN" required:"true" description:"Runner token"`
	TLSCAFile   string `toml:"tls-ca-file,omitempty" json:"tls-ca-file" long:"tls-ca-file" env:"CI_SERVER_TLS_CA_FILE" description:"File containing the certificates to verify the peer when using HTTPS"`
	TLSCertFile string `toml:"tls-cert-file,omitempty" json:"tls-cert-file" long:"tls-cert-file" env:"CI_SERVER_TLS_CERT_FILE" description:"File containing certificate for TLS client auth when using HTTPS"`
	TLSKeyFile  string `toml:"tls-key-file,omitempty" json:"tls-key-file" long:"tls-key-file" env:"CI_SERVER_TLS_KEY_FILE" description:"File containing private key for TLS client auth when using HTTPS"`
}

//nolint:lll
type CacheGCSCredentials struct {
	AccessID   string `toml:"AccessID,omitempty" long:"access-id" env:"CACHE_GCS_ACCESS_ID" description:"ID of GCP Service Account used to access the storage"`
	PrivateKey string `toml:"PrivateKey,omitempty" long:"private-key" env:"CACHE_GCS_PRIVATE_KEY" description:"Private key used to sign GCS requests"`
}

//nolint:lll
type CacheGCSConfig struct {
	CacheGCSCredentials
	CredentialsFile string `toml:"CredentialsFile,omitempty" long:"credentials-file" env:"GOOGLE_APPLICATION_CREDENTIALS" description:"File with GCP credentials, containing AccessID and PrivateKey"`
	BucketName      string `toml:"BucketName,omitempty" long:"bucket-name" env:"CACHE_GCS_BUCKET_NAME" description:"Name of the bucket where cache will be stored"`
}

//nolint:lll
type CacheS3Config struct {
	ServerAddress  string `toml:"ServerAddress,omitempty" long:"server-address" env:"CACHE_S3_SERVER_ADDRESS" description:"A host:port to the used S3-compatible server"`
	AccessKey      string `toml:"AccessKey,omitempty" long:"access-key" env:"CACHE_S3_ACCESS_KEY" description:"S3 Access Key"`
	SecretKey      string `toml:"SecretKey,omitempty" long:"secret-key" env:"CACHE_S3_SECRET_KEY" description:"S3 Secret Key"`
	BucketName     string `toml:"BucketName,omitempty" long:"bucket-name" env:"CACHE_S3_BUCKET_NAME" description:"Name of the bucket where cache will be stored"`
	BucketLocation string `toml:"BucketLocation,omitempty" long:"bucket-location" env:"CACHE_S3_BUCKET_LOCATION" description:"Name of S3 region"`
	Insecure       bool   `toml:"Insecure,omitempty" long:"insecure" env:"CACHE_S3_INSECURE" description:"Use insecure mode (without https)"`
}

//nolint:lll
type CacheAzureCredentials struct {
	AccountName string `toml:"AccountName,omitempty" long:"account-name" env:"CACHE_AZURE_ACCOUNT_NAME" description:"Account name for Azure Blob Storage"`
	AccountKey  string `toml:"AccountKey,omitempty" long:"account-key" env:"CACHE_AZURE_ACCOUNT_KEY" description:"Access key for Azure Blob Storage"`
}

//nolint:lll
type CacheAzureConfig struct {
	CacheAzureCredentials
	ContainerName string `toml:"ContainerName,omitempty" long:"container-name" env:"CACHE_AZURE_CONTAINER_NAME" description:"Name of the Azure container where cache will be stored"`
	StorageDomain string `toml:"StorageDomain,omitempty" long:"storage-domain" env:"CACHE_AZURE_STORAGE_DOMAIN" description:"Domain name of the Azure storage (e.g. blob.core.windows.net)"`
}

//nolint:lll
type CacheConfig struct {
	Type   string `toml:"Type,omitempty" long:"type" env:"CACHE_TYPE" description:"Select caching method"`
	Path   string `toml:"Path,omitempty" long:"path" env:"CACHE_PATH" description:"Name of the path to prepend to the cache URL"`
	Shared bool   `toml:"Shared,omitempty" long:"shared" env:"CACHE_SHARED" description:"Enable cache sharing between runners."`

	S3    *CacheS3Config    `toml:"s3,omitempty" json:"s3" namespace:"s3"`
	GCS   *CacheGCSConfig   `toml:"gcs,omitempty" json:"gcs" namespace:"gcs"`
	Azure *CacheAzureConfig `toml:"azure,omitempty" json:"azure" namespace:"azure"`
}

//nolint:lll
type RunnerSettings struct {
	Executor  string `toml:"executor" json:"executor" long:"executor" env:"RUNNER_EXECUTOR" required:"true" description:"Select executor (anka or ssh)"`
	BuildsDir string `toml:"builds_dir,omitempty" json:"builds_dir" long:"builds-dir" env:"RUNNER_BUILDS_DIR" description:"Directory where builds are stored"`
	CacheDir  string `toml:"cache_dir,omitempty" json:"cache_dir" long:"cache-dir" env:"RUNNER_CACHE_DIR" description:"Directory where build cache is stored"`
	CloneURL  string `toml:"clone_url,omitempty" json:"clone_url" long:"clone-url" env:"CLONE_URL" description:"Overwrite the default URL used to clone or fetch the git ref"`

	Environment     []string `toml:"environment,omitempty" json:"environment" long:"env" env:"RUNNER_ENV" description:"Custom environment variables injected to build environment"`
	PreCloneScript  string   `toml:"pre_clone_script,omitempty" json:"pre_clone_script" long:"pre-clone-script" env:"RUNNER_PRE_CLONE_SCRIPT" description:"Runner-specific command script executed before code is pulled"`
	PreBuildScript  string   `toml:"pre_build_script,omitempty" json:"pre_build_script" long:"pre-build-script" env:"RUNNER_PRE_BUILD_SCRIPT" description:"Runner-specific command script executed after code is pulled, just before build executes"`
	PostBuildScript string   `toml:"post_build_script,omitempty" json:"post_build_script" long:"post-build-script" env:"RUNNER_POST_BUILD_SCRIPT" description:"Runner-specific command script executed after code is pulled and just after build executes"`

	DebugTraceDisabled bool `toml:"debug_trace_disabled,omitempty" json:"debug_trace_disabled" long:"debug-trace-disabled" env:"RUNNER_DEBUG_TRACE_DISABLED" description:"When set to true Runner will disable the possibility of using the CI_DEBUG_TRACE feature"`

	Shell          string           `toml:"shell,omitempty" json:"shell" long:"shell" env:"RUNNER_SHELL" description:"Select bash, cmd or powershell"`
	CustomBuildDir *CustomBuildDir  `toml:"custom_build_dir,omitempty" json:"custom_build_dir" group:"custom build dir configuration" namespace:"custom_build_dir"`
	Referees       *referees.Config `toml:"referees,omitempty" json:"referees" group:"referees configuration" namespace:"referees"`
	Cache          *CacheConfig     `toml:"cache,omitempty" json:"cache" group:"cache configuration" namespace:"cache"`

	PreparationRetries int `toml:"preparation_retries,omitzero" json:"preparation_retries" long:"preparation-retries" env:"PREPARATION_RETRIES" description:"Set the amount of preparation retries for a job"`

	SSH *ssh.Config `toml:"ssh,omitempty" json:"ssh" group:"ssh executor" namespace:"ssh"`
	// Docker     *DockerConfig     `toml:"docker,omitempty" json:"docker" group:"docker executor" namespace:"docker"`
	// Parallels  *ParallelsConfig  `toml:"parallels,omitempty" json:"parallels" group:"parallels executor" namespace:"parallels"`
	// VirtualBox *VirtualBoxConfig `toml:"virtualbox,omitempty" json:"virtualbox" group:"virtualbox executor" namespace:"virtualbox"`
	// Machine    *DockerMachine    `toml:"machine,omitempty" json:"machine" group:"docker machine provider" namespace:"machine"`
	// Kubernetes *KubernetesConfig `toml:"kubernetes,omitempty" json:"kubernetes" group:"kubernetes executor" namespace:"kubernetes"`
	// Custom     *CustomConfig     `toml:"custom,omitempty" json:"custom" group:"custom executor" namespace:"custom"`
	Anka *AnkaConfig `toml:"anka,omitempty" json:"anka" group:"anka executor" namespace:"anka"`
}

//nolint:lll
type RunnerConfig struct {
	Name               string `toml:"name" json:"name" short:"name" long:"description" env:"RUNNER_NAME" description:"Runner name"`
	Limit              int    `toml:"limit,omitzero" json:"limit" long:"limit" env:"RUNNER_LIMIT" description:"Maximum number of builds processed by this runner"`
	OutputLimit        int    `toml:"output_limit,omitzero" long:"output-limit" env:"RUNNER_OUTPUT_LIMIT" description:"Maximum build trace size in kilobytes"`
	RequestConcurrency int    `toml:"request_concurrency,omitzero" long:"request-concurrency" env:"RUNNER_REQUEST_CONCURRENCY" description:"Maximum concurrency for job requests"`

	RunnerCredentials
	RunnerSettings
}

//nolint:lll
type SessionServer struct {
	ListenAddress    string `toml:"listen_address,omitempty" json:"listen_address" description:"Address that the runner will communicate directly with"`
	AdvertiseAddress string `toml:"advertise_address,omitempty" json:"advertise_address" description:"Address the runner will expose to the world to connect to the session server"`
	SessionTimeout   int    `toml:"session_timeout,omitempty" json:"session_timeout" description:"How long a terminal session can be active after a build completes, in seconds"`
}

//nolint:lll
type Config struct {
	ListenAddress string          `toml:"listen_address,omitempty" json:"listen_address"`
	SessionServer SessionServer   `toml:"session_server,omitempty" json:"session_server"`
	Concurrent    int             `toml:"concurrent" json:"concurrent"`
	CheckInterval int             `toml:"check_interval" json:"check_interval" description:"Define active checking interval of jobs"`
	LogLevel      *string         `toml:"log_level" json:"log_level" description:"Define log level (one of: panic, fatal, error, warning, info, debug)"`
	LogFormat     *string         `toml:"log_format" json:"log_format" description:"Define log format (one of: runner, text, json)"`
	User          string          `toml:"user,omitempty" json:"user"`
	Runners       []*RunnerConfig `toml:"runners" json:"runners"`
	SentryDSN     *string         `toml:"sentry_dsn"`
	ModTime       time.Time       `toml:"-"`
	Loaded        bool            `toml:"-"`
}

//nolint:lll
type CustomBuildDir struct {
	Enabled bool `toml:"enabled,omitempty" json:"enabled" long:"enabled" env:"CUSTOM_BUILD_DIR_ENABLED" description:"Enable job specific build directories"`
}

func (c *CacheS3Config) ShouldUseIAMCredentials() bool {
	return c.ServerAddress == "" || c.AccessKey == "" || c.SecretKey == ""
}

func (c *CacheConfig) GetPath() string {
	return c.Path
}

func (c *CacheConfig) GetShared() bool {
	return c.Shared
}

func (c *SessionServer) GetSessionTimeout() time.Duration {
	if c.SessionTimeout > 0 {
		return time.Duration(c.SessionTimeout) * time.Second
	}

	return DefaultSessionTimeout
}

func (c *RunnerCredentials) GetURL() string {
	return c.URL
}

func (c *RunnerCredentials) GetTLSCAFile() string {
	return c.TLSCAFile
}

func (c *RunnerCredentials) GetTLSCertFile() string {
	return c.TLSCertFile
}

func (c *RunnerCredentials) GetTLSKeyFile() string {
	return c.TLSKeyFile
}

func (c *RunnerCredentials) GetToken() string {
	return c.Token
}

func (c *RunnerCredentials) ShortDescription() string {
	return helpers.ShortenToken(c.Token)
}

func (c *RunnerCredentials) UniqueID() string {
	return c.URL + c.Token
}

func (c *RunnerCredentials) Log() *logrus.Entry {
	if c.ShortDescription() != "" {
		return logrus.WithField("runner", c.ShortDescription())
	}
	return logrus.WithFields(logrus.Fields{})
}

func (c *RunnerCredentials) SameAs(other *RunnerCredentials) bool {
	return c.URL == other.URL && c.Token == other.Token
}

func (c *RunnerConfig) String() string {
	return fmt.Sprintf("%v url=%v token=%v executor=%v", c.Name, c.URL, c.Token, c.Executor)
}

func (c *RunnerConfig) GetRequestConcurrency() int {
	if c.RequestConcurrency <= 0 {
		return 1
	}
	return c.RequestConcurrency
}

func (c *RunnerConfig) GetVariables() JobVariables {
	variables := JobVariables{
		{Key: "CI_RUNNER_SHORT_TOKEN", Value: c.ShortDescription(), Public: true, Internal: true, File: false},
	}

	for _, environment := range c.Environment {
		if variable, err := ParseVariable(environment); err == nil {
			variable.Internal = true
			variables = append(variables, variable)
		}
	}

	return variables
}

// DeepCopy attempts to make a deep clone of the object
func (c *RunnerConfig) DeepCopy() (*RunnerConfig, error) {
	var r RunnerConfig

	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("serialization of runner config failed: %w", err)
	}

	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return nil, fmt.Errorf("deserialization of runner config failed: %w", err)
	}

	return &r, err
}

func NewConfig() *Config {
	return &Config{
		Concurrent: 1,
		SessionServer: SessionServer{
			SessionTimeout: int(DefaultSessionTimeout.Seconds()),
		},
	}
}

func (c *Config) StatConfig(configFile string) error {
	_, err := os.Stat(configFile)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) LoadConfig(configFile string) error {
	info, err := os.Stat(configFile)

	// permission denied is soft error
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if _, err = toml.DecodeFile(configFile, c); err != nil {
		return err
	}

	c.ModTime = info.ModTime()
	c.Loaded = true
	return nil
}

func (c *Config) SaveConfig(configFile string) error {
	var newConfig bytes.Buffer
	newBuffer := bufio.NewWriter(&newConfig)

	if err := toml.NewEncoder(newBuffer).Encode(c); err != nil {
		logrus.Fatalf("Error encoding TOML: %s", err)
		return err
	}

	if err := newBuffer.Flush(); err != nil {
		return err
	}

	// create directory to store configuration
	err := os.MkdirAll(filepath.Dir(configFile), 0700)
	if err != nil {
		return err
	}

	// write config file
	if err := ioutil.WriteFile(configFile, newConfig.Bytes(), 0600); err != nil {
		return err
	}

	c.Loaded = true
	return nil
}

func (c *Config) GetCheckInterval() time.Duration {
	if c.CheckInterval > 0 {
		return time.Duration(c.CheckInterval) * time.Second
	}
	return CheckInterval
}
