package common

import (
	"fmt"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/featureflags"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/process"
)

func TestCacheS3Config_ShouldUseIAMCredentials(t *testing.T) {
	tests := map[string]struct {
		s3                     CacheS3Config
		shouldUseIAMCredential bool
	}{
		"Everything is empty": {
			s3: CacheS3Config{
				ServerAddress:  "",
				AccessKey:      "",
				SecretKey:      "",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"Both AccessKey & SecretKey are empty": {
			s3: CacheS3Config{
				ServerAddress:  "s3.amazonaws.com",
				AccessKey:      "",
				SecretKey:      "",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"SecretKey is empty": {
			s3: CacheS3Config{
				ServerAddress:  "s3.amazonaws.com",
				AccessKey:      "TOKEN",
				SecretKey:      "",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"AccessKey is empty": {
			s3: CacheS3Config{
				ServerAddress:  "s3.amazonaws.com",
				AccessKey:      "",
				SecretKey:      "TOKEN",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"ServerAddress is empty": {
			s3: CacheS3Config{
				ServerAddress:  "",
				AccessKey:      "TOKEN",
				SecretKey:      "TOKEN",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"ServerAddress & AccessKey are empty": {
			s3: CacheS3Config{
				ServerAddress:  "",
				AccessKey:      "",
				SecretKey:      "TOKEN",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"ServerAddress & SecretKey are empty": {
			s3: CacheS3Config{
				ServerAddress:  "",
				AccessKey:      "TOKEN",
				SecretKey:      "",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: true,
		},
		"Nothing is empty": {
			s3: CacheS3Config{
				ServerAddress:  "s3.amazonaws.com",
				AccessKey:      "TOKEN",
				SecretKey:      "TOKEN",
				BucketName:     "name",
				BucketLocation: "us-east-1a",
			},
			shouldUseIAMCredential: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.shouldUseIAMCredential, tt.s3.ShouldUseIAMCredentials())
		})
	}
}

func TestConfigParse(t *testing.T) {
	tests := map[string]struct {
		config         string
		validateConfig func(t *testing.T, config *Config)
		expectedErr    string
	}{
		"parse AnkaService int as name not allowed": {
			config: `
				[[runners]]
					name = 123`,
			expectedErr: "toml: cannot load TOML value of type int64 into a Go string",
		},
		"parse AnkaService int as controller_address": {
			config: `
				[[runners]]
					name = "localhost-shared"
				[runners.anka]
					controller_address = 1`,
			expectedErr: "toml: cannot load TOML value of type int64 into a Go string",
		},
		"parse AnkaService url": {
			config: `
			[[runners]]
				name = "localhost-shared"
				url = "http://anka-gitlab-ce:8084/"
				token = "LCQrsBLsB86DQRe8Lpo6"
				executor = "anka"
				clone_url = "http://anka-gitlab-ce:8084"
				preparation_retries = 1
				[runners.custom_build_dir]
				[runners.cache]
					[runners.cache.s3]
					[runners.cache.gcs]
				[runners.ssh]
					user = "anka"
					password = "admin"
				[runners.anka]
					controller_address = "https://127.0.0.1:8080/"
					template_uuid = "c0847bc9-5d2d-4dbc-ba6a-240f7ff08032"
					tag = "base:port-forward-22:brew-git:gitlab"
					root_ca_path = "/Users/user1/anka-ca-crt.pem"
					cert_path = "/Users/user1/gitlab-crt.pem"
					key_path = "/Users/user1/gitlab-key.pem"
					keep_alive_on_error = false
					skip_tls_verification = false`,
			validateConfig: func(t *testing.T, config *Config) {
				require.Equal(t, 1, len(config.Runners))
				assert.Equal(t, "localhost-shared", config.Runners[0].Name)
				assert.Equal(t, 1, config.Runners[0].PreparationRetries)
				assert.Equal(t, false, config.Runners[0].Anka.SkipTLSVerification)
				assert.Equal(t, "anka", config.Runners[0].SSH.User)
			},
		},
	}

	for tn, tt := range tests {
		fmt.Println(fmt.Sprintf("%s%s%s", helpers.ANSI_BOLD_CYAN, "------------------", helpers.ANSI_RESET))
		fmt.Println(fmt.Sprintf("%s%s %s%s", helpers.ANSI_BOLD_CYAN, "Testing:", tn, helpers.ANSI_RESET))
		t.Run(tn, func(t *testing.T) {
			cfg := NewConfig()
			_, err := toml.Decode(tt.config, cfg)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				return
			}

			assert.NoError(t, err)
			if tt.validateConfig != nil {
				tt.validateConfig(t, cfg)
			}
		})
	}
}

func TestRunnerSettings_GetGracefulKillTimeout_GetForceKillTimeout(t *testing.T) {
	tests := map[string]struct {
		config                      RunnerSettings
		expectedGracefulKillTimeout time.Duration
		expectedForceKillTimeout    time.Duration
	}{
		"undefined": {
			config:                      RunnerSettings{},
			expectedGracefulKillTimeout: process.GracefulTimeout,
			expectedForceKillTimeout:    process.KillTimeout,
		},
		"timeouts lower than 0": {
			config: RunnerSettings{
				GracefulKillTimeout: func(i int) *int { return &i }(-10),
				ForceKillTimeout:    func(i int) *int { return &i }(-10),
			},
			expectedGracefulKillTimeout: process.GracefulTimeout,
			expectedForceKillTimeout:    process.KillTimeout,
		},
		"timeouts greater than 0": {
			config: RunnerSettings{
				GracefulKillTimeout: func(i int) *int { return &i }(30),
				ForceKillTimeout:    func(i int) *int { return &i }(15),
			},
			expectedGracefulKillTimeout: 30 * time.Second,
			expectedForceKillTimeout:    15 * time.Second,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			assert.Equal(t, tt.expectedGracefulKillTimeout, tt.config.GetGracefulKillTimeout())
			assert.Equal(t, tt.expectedForceKillTimeout, tt.config.GetForceKillTimeout())
		})
	}
}

func TestStringOrArray_UnmarshalTOML(t *testing.T) {
	tests := map[string]struct {
		toml           string
		expectedResult StringOrArray
		expectedErr    bool
	}{
		"no fields": {
			toml:           "",
			expectedResult: nil,
			expectedErr:    false,
		},
		"empty string_or_array": {
			toml:           `string_or_array = ""`,
			expectedResult: StringOrArray{""},
			expectedErr:    false,
		},
		"string": {
			toml:           `string_or_array = "always"`,
			expectedResult: StringOrArray{"always"},
			expectedErr:    false,
		},
		"slice with invalid single value": {
			toml:        `string_or_array = 10`,
			expectedErr: true,
		},
		"valid slice with multiple values": {
			toml:           `string_or_array = ["unknown", "always"]`,
			expectedResult: StringOrArray{"unknown", "always"},
			expectedErr:    false,
		},
		"slice with mixed values": {
			toml:        `string_or_array = ["unknown", 10]`,
			expectedErr: true,
		},
		"slice with invalid values": {
			toml:        `string_or_array = [true, false]`,
			expectedErr: true,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			type Config struct {
				StringOrArray StringOrArray `toml:"string_or_array"`
			}

			var result Config
			_, err := toml.Decode(tt.toml, &result)

			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result.StringOrArray)
		})
	}
}

func TestRunnerSettings_IsFeatureFlagOn(t *testing.T) {
	tests := map[string]struct {
		featureFlags  map[string]bool
		name          string
		expectedValue bool
	}{
		"feature flag not configured": {
			featureFlags:  map[string]bool{},
			name:          t.Name(),
			expectedValue: false,
		},
		"feature flag not configured but feature flag default is true": {
			featureFlags:  map[string]bool{},
			name:          featureflags.UseDirectDownload,
			expectedValue: true,
		},
		"feature flag on": {
			featureFlags: map[string]bool{
				t.Name(): true,
			},
			name:          t.Name(),
			expectedValue: true,
		},
		"feature flag off": {
			featureFlags: map[string]bool{
				featureflags.UseDirectDownload: false,
			},
			name:          t.Name(),
			expectedValue: false,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			cfg := RunnerConfig{
				RunnerSettings: RunnerSettings{
					FeatureFlags: tt.featureFlags,
				},
			}

			on := cfg.IsFeatureFlagOn(tt.name)
			assert.Equal(t, tt.expectedValue, on)
		})
	}
}
