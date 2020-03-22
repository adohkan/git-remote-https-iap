package main

import (
	"fmt"
	_url "net/url"
	"os"
	"strings"
	"time"

	"github.com/adohkan/git-remote-https-iap/internal/git"
	"github.com/adohkan/git-remote-https-iap/internal/iap"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	BinaryName = "git-remote-https+iap"
)

var (
	version string

	rootCmd = &cobra.Command{
		Use:   fmt.Sprintf("%s remote url", BinaryName),
		Short: "git-remote-helper that handles authentication for GCP Identity Aware Proxy",
		Args:  cobra.ExactArgs(2),
		Run:   execute,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version number",
		Run:   printVersion,
	}

	installProtocolCmd = &cobra.Command{
		Use:   "install",
		Short: "Install protocol in Git config",
		Run:   installGitProtocol,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(installProtocolCmd)

	// set default log level
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) {
	remote, url := args[0], args[1]
	log.Debug().Msgf("%s %s %s", BinaryName, remote, url)

	handleIAPAuthCookieFor(url)
	git.PassThruRemoteHTTPSHelper(remote, url)
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("%s %s\n", BinaryName, version)
}

func installGitProtocol(cmd *cobra.Command, args []string) {
	p := strings.TrimLeft(BinaryName, "git-remote-")
	git.InstallProtocol(p)
	log.Info().Msgf("%s protocol configured in git!", p)
}

func handleIAPAuthCookieFor(url string) {
	// All our work will be based on the basedomain of the provided URL
	// as IAP would be setup for the whole domain.
	url, err := toHTTPSBaseDomain(url)
	if err != nil {
		log.Error().Msgf("Could not convert %s in https://: %s", url, err)
	}

	log.Debug().Msgf("Manage IAP auth for %s", url)

	cookie, err := iap.ReadCookie(url)
	switch {
	case err != nil:
		log.Debug().Msgf("could not read IAP cookie for %s: %s", url, err.Error())
		cookie, err = iap.NewCookie(url)
	case cookie.Expired():
		log.Debug().Msgf("IAP cookie for %s has expired", url)
		cookie, err = iap.NewCookie(url)
	case !cookie.Expired():
		log.Debug().Msgf("IAP Cookie still valid until %s", time.Unix(cookie.Claims.ExpiresAt, 0))
	}

	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func toHTTPSBaseDomain(addr string) (string, error) {
	u, err := _url.Parse(addr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s", u.Host), nil
}
