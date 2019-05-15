package helpers

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type HealthCheckCommand struct{}

func (c *HealthCheckCommand) Execute(ctx *cli.Context) {
	var addr, port string

	for _, e := range os.Environ() {
		parts := strings.Split(e, "=")

		if len(parts) != 2 {
			continue
		} else if strings.HasSuffix(parts[0], "_TCP_ADDR") {
			addr = parts[1]
		} else if strings.HasSuffix(parts[0], "_TCP_PORT") {
			port = parts[1]
		}
	}

	if addr == "" || port == "" {
		logrus.Fatalln("No HOST or PORT found")
	}

	fmt.Fprintf(os.Stdout, "waiting for TCP connection to %s:%s...", addr, port)

	for {
		conn, err := net.Dial("tcp", net.JoinHostPort(addr, port))
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		conn.Close()
		return
	}
}

func init() {
	common.RegisterCommand2("health-check", "check health for a specific address", &HealthCheckCommand{})
}
