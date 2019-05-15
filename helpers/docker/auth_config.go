package docker_helpers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/homedir"
)

// DefaultDockerRegistry is the name of the index
const DefaultDockerRegistry = "docker.io"

// EncodeAuthConfig constructs a token from an AuthConfig, suitable for
// authorizing against the Docker API with.
func EncodeAuthConfig(authConfig *types.AuthConfig) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(authConfig); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

// SplitDockerImageName breaks a reposName into an index name and remote name
func SplitDockerImageName(reposName string) (string, string) {
	nameParts := strings.SplitN(reposName, "/", 2)
	var indexName, remoteName string
	if len(nameParts) == 1 || (!strings.Contains(nameParts[0], ".") &&
		!strings.Contains(nameParts[0], ":") && nameParts[0] != "localhost") {
		// This is a Docker Index repos (ex: samalba/hipache or ubuntu)
		// 'docker.io'
		indexName = DefaultDockerRegistry
		remoteName = reposName
	} else {
		indexName = nameParts[0]
		remoteName = nameParts[1]
	}

	if indexName == "index."+DefaultDockerRegistry {
		indexName = DefaultDockerRegistry
	}
	return indexName, remoteName
}

var HomeDirectory = homedir.Get()

func ReadDockerAuthConfigsFromHomeDir(userName string) (map[string]types.AuthConfig, error) {
	homeDir := HomeDirectory

	if userName != "" {
		u, err := user.Lookup(userName)
		if err != nil {
			return nil, err
		}
		homeDir = u.HomeDir
	}

	if homeDir == "" {
		return nil, fmt.Errorf("Failed to get home directory")
	}

	p := path.Join(homeDir, ".docker", "config.json")

	r, err := os.Open(p)
	defer r.Close()

	if err != nil {
		p := path.Join(homeDir, ".dockercfg")
		r, err = os.Open(p)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	if r == nil {
		return make(map[string]types.AuthConfig), nil
	}

	return ReadAuthConfigsFromReader(r)
}

func ReadAuthConfigsFromReader(r io.Reader) (map[string]types.AuthConfig, error) {
	config := &configfile.ConfigFile{}

	if err := config.LoadFromReader(r); err != nil {
		return nil, err
	}

	auths := make(map[string]types.AuthConfig)
	addAll(auths, config.AuthConfigs)

	if config.CredentialsStore != "" {
		authsFromCredentialsStore, err := readAuthConfigsFromCredentialsStore(config)
		if err != nil {
			return nil, err
		}
		addAll(auths, authsFromCredentialsStore)
	}

	return auths, nil
}

func readAuthConfigsFromCredentialsStore(config *configfile.ConfigFile) (map[string]types.AuthConfig, error) {
	store := credentials.NewNativeStore(config, config.CredentialsStore)

	newAuths, err := store.GetAll()

	if err != nil {
		return nil, err
	}

	return newAuths, nil
}

func addAll(to, from map[string]types.AuthConfig) {
	for reg, ac := range from {
		to[reg] = ac
	}
}

// ResolveDockerAuthConfig taken from: https://github.com/docker/docker/blob/master/registry/auth.go
func ResolveDockerAuthConfig(indexName string, configs map[string]types.AuthConfig) *types.AuthConfig {
	if configs == nil {
		return nil
	}

	convertToHostname := func(url string) string {
		stripped := url
		if strings.HasPrefix(url, "http://") {
			stripped = strings.Replace(url, "http://", "", 1)
		} else if strings.HasPrefix(url, "https://") {
			stripped = strings.Replace(url, "https://", "", 1)
		}

		nameParts := strings.SplitN(stripped, "/", 2)
		if nameParts[0] == "index."+DefaultDockerRegistry {
			return DefaultDockerRegistry
		}
		return nameParts[0]
	}

	// Maybe they have a legacy config file, we will iterate the keys converting
	// them to the new format and testing
	for registry, authConfig := range configs {
		if indexName == convertToHostname(registry) {
			return &authConfig
		}
	}

	// When all else fails, return an empty auth config
	return nil
}
