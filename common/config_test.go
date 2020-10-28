package common

import (
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
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

func TestService_ToImageDefinition(t *testing.T) {
	tests := map[string]struct {
		service       Service
		expectedImage Image
	}{
		"empty service": {
			service:       Service{},
			expectedImage: Image{},
		},
		"only name": {
			service:       Service{Name: "name"},
			expectedImage: Image{Name: "name"},
		},
		"only alias": {
			service:       Service{Alias: "alias"},
			expectedImage: Image{Alias: "alias"},
		},
		"name and alias": {
			service:       Service{Name: "name", Alias: "alias"},
			expectedImage: Image{Name: "name", Alias: "alias"},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			assert.Equal(t, tt.expectedImage, tt.service.ToImageDefinition())
		})
	}
}
