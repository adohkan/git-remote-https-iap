package git

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	// GitBinary defines the name of the git client's binary on the system
	GitBinary = "git"
)

// ConfigGetURLMatch call 'git config --get-urlmatch' underneath
func ConfigGetURLMatch(key, url string) string {
	var stdout bytes.Buffer

	args := []string{"config", "--get-urlmatch", key, url}
	cmd := exec.Command(GitBinary, args...)
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		log.Fatal().Msgf("ConfigGetURLMatch - could not read config '%s' for '%s' (%s)", key, url, err)
	}

	return strings.TrimSpace(string(stdout.Bytes()))
}
