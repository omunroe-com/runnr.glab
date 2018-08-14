package shells

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

func TestBash_CommandShellEscapes(t *testing.T) {
	writer := &BashWriter{}
	writer.Command("foo", "x&(y)")

	assert.Equal(t, `$'foo' "x&(y)"`+"\n", writer.String())
}

func TestBash_IfCmdShellEscapes(t *testing.T) {
	writer := &BashWriter{}
	writer.IfCmd("foo", "x&(y)")

	assert.Equal(t, `if $'foo' "x&(y)" >/dev/null 2>/dev/null; then`+"\n", writer.String())
}

func TestBashShell_GetConfiguration(t *testing.T) {
	testCases := map[string]struct {
		shellType         common.ShellType
		expectedCommand   string
		expectedArguments []string
		expectedLogin     bool
	}{
		"normal shell": {
			shellType:         common.NormalShell,
			expectedCommand:   "bash",
			expectedArguments: []string{},
			expectedLogin:     false,
		},
		"login shell": {
			shellType:       common.LoginShell,
			expectedCommand: "bash",
			expectedArguments: []string{
				"--login",
			},
			expectedLogin: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			shell := &BashShell{
				Shell: "bash",
			}

			configuration, err := shell.GetConfiguration(common.ShellScriptInfo{
				Type: testCase.shellType,
			})
			require.NoError(t, err)

			assert.Equal(t, testCase.expectedCommand, configuration.Command)

			if len(testCase.expectedArguments) == 0 {
				assert.Empty(t, configuration.Arguments)
			} else {
				assert.Equal(t, testCase.expectedArguments, configuration.Arguments)
			}

			assert.Contains(t, configuration.DockerCommand, "sh")
			assert.Contains(t, configuration.DockerCommand, "-c")

			require.Len(t, configuration.DockerCommand, 3)

			if testCase.expectedLogin {
				assert.Contains(t, configuration.DockerCommand[2], "exec /usr/bin/bash --login")
			} else {
				assert.NotContains(t, configuration.DockerCommand[2], "exec /usr/bin/bash --login")
			}
		})
	}
}
