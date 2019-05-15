package helpers

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
)

func newCacheInitTestApp() *cli.App {
	cmd := &CacheInitCommand{}
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Commands = append(app.Commands, cli.Command{
		Name:   "cache-init",
		Action: cmd.Execute,
	})

	return app
}

func TestCacheInit(t *testing.T) {
	// Specifically test a dir name with spaces.
	dir, err := ioutil.TempDir("", "Test Cache Chmod")
	require.NoError(t, err)

	defer os.Remove(dir)

	// Make sure that the mode is not the expect 0777.
	err = os.Chmod(dir, 0600)
	require.NoError(t, err)

	// Start a new cli with the arguments for the command.
	args := os.Args[0:1]
	args = append(args, "cache-init", dir)

	err = newCacheInitTestApp().Run(args)
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)

	assert.Equal(t, os.ModeDir+os.ModePerm, info.Mode())
}

func TestCacheInit_NoArguments(t *testing.T) {
	removeHook := helpers.MakeFatalToPanic()
	defer removeHook()

	args := os.Args[0:1]
	args = append(args, "cache-init")

	assert.Panics(t, func() {
		newCacheInitTestApp().Run(args)
	})
}
