package formatter

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestConfigContextFormatWrite(t *testing.T) {
	// Check default output format (verbose and non-verbose mode) for table headers
	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{Context{Format: NewConfigFormat("table", false)},
			`ID                  NAME                CREATED                  UPDATED
1                   passwords           Less than a second ago   Less than a second ago
2                   id_rsa              Less than a second ago   Less than a second ago
`},
		{Context{Format: NewConfigFormat("table {{.Name}}", true)},
			`NAME
passwords
id_rsa
`},
		{Context{Format: NewConfigFormat("{{.ID}}-{{.Name}}", false)},
			`1-passwords
2-id_rsa
`},
	}

	configs := []swarm.Config{
		{ID: "1",
			Meta: swarm.Meta{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "passwords"}}},
		{ID: "2",
			Meta: swarm.Meta{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "id_rsa"}}},
	}
	for _, testcase := range cases {
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		if err := ConfigWrite(testcase.context, configs); err != nil {
			assert.ErrorContains(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(out.String(), testcase.expected))
		}
	}
}
