package git

import (
	"bytes"
	"fmt"
	_url "net/url"
	"os"
	"os/exec"
	"regexp"
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

func StoreCredentials(protocol, host, username, password string) error {
	var stdin bytes.Buffer

	cmd := exec.Command(GitBinary, "credential-store", "store")
	// see: https://git-scm.com/docs/git-credential
	params := fmt.Sprintf("protocol=%s\nhost=%s\nusername=%s\npassword=%s\n", protocol, host, username, password)
	if _, err := stdin.Write([]byte(params)); err != nil {
		return err
	}
	cmd.Stdin = &stdin
	res := cmd.Run()
	if res == nil {
		log.Debug().Msgf("StoreCredentials - credentials saved for protocol=%s,host=%s,username=%s", protocol, host, username)
	}
	return res
}

func GetCredentials(protocol, host, username string) (string, error) {
	var stdin, stdout bytes.Buffer

	cmd := exec.Command(GitBinary, "credential-store", "get")
	// see: https://git-scm.com/docs/git-credential
	params := fmt.Sprintf("protocol=%s\nhost=%s\nusername=%s\n", protocol, host, username)
	stdin.Write([]byte(params))
	cmd.Stdin = &stdin
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	match := regexp.MustCompile("password=(.*)").FindStringSubmatch(string(stdout.Bytes()))
	if match != nil {
		log.Debug().Msgf("GetCredentials - found credentials for protocol=%s,host=%s,username=%s", protocol, host, username)
		return match[1], nil
	}

	return "", fmt.Errorf("GetCredentials - not found for protocol=%s,host=%s,username=%s", protocol, host, username)
}
