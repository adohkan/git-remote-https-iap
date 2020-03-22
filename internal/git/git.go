package git

import (
	"bytes"
	_url "net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"

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

func PassThruRemoteHTTPSHelper(remote, url string) {
	u, err := _url.Parse(url)
	if err != nil {
		log.Fatal().Msgf("passThruRemoteHTTPSHelper - could not parse %s: %s", url, err.Error())
	}
	u.Scheme = "https"
	args := []string{"git", "remote-https", remote, u.String()}
	log.Debug().Msgf("passThruRemoteHTTPSHelper exec: %v", args)

	binary, err := exec.LookPath(GitBinary)
	if err != nil {
		log.Fatal().Msgf("passThruRemoteHTTPSHelper - %s", err.Error())
	}

	env := os.Environ()
	if err := syscall.Exec(binary, args, env); err != nil {
		log.Fatal().Msgf("passThruRemoteHTTPSHelper - %s", err.Error())
	}
}
