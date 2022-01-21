package main

import (
	"fmt"
	_url "net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adohkan/git-remote-https-iap/internal/git"
	"github.com/adohkan/git-remote-https-iap/internal/iap"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	// DebugEnvVariable is the name of the environment variable that needs to be set in order to enable debug logging
	DebugEnvVariable = "GIT_IAP_VERBOSE"
)

var (
	binaryName = os.Args[0]
	version    string

	// only used in configureCmd
	repoURL, helperID, helperSecret, clientID string

	rootCmd = &cobra.Command{
		Use:   fmt.Sprintf("%s remote url", binaryName),
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

	configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Configure IAP for a given repository",
		Run:   configureIAP,
	}

	checkCmd = &cobra.Command{
		Use:   "check remote url",
		Short: "Refresh token for remote url if needed, then exit",
		Run:   check,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(installProtocolCmd)
	rootCmd.AddCommand(checkCmd)

	configureCmd.Flags().StringVar(&repoURL, "repoURL", "", "URL of the git repository to configure (required)")
	configureCmd.MarkFlagRequired("repoURL")
	configureCmd.Flags().StringVar(&helperID, "helperID", "", "OAuth Client ID for the helper (required)")
	configureCmd.MarkFlagRequired("helperID")
	configureCmd.Flags().StringVar(&helperSecret, "helperSecret", "", "OAuth Client Secret for the helper (required)")
	configureCmd.MarkFlagRequired("helperSecret")
	configureCmd.Flags().StringVar(&clientID, "clientID", "", "OAuth Client ID of the IAP instance (required)")
	configureCmd.MarkFlagRequired("clientID")
	rootCmd.AddCommand(configureCmd)

	// set log level
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	if debug, _ := strconv.ParseBool(os.Getenv(DebugEnvVariable)); debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) {
	remote, url := args[0], args[1]
	log.Debug().Msgf("%s %s %s", binaryName, remote, url)

	c := handleIAPAuthCookieFor(url)
	git.PassThruRemoteHTTPSHelper(remote, url, c.Token.Raw)
}

func check(cmd *cobra.Command, args []string) {
	remote, url := args[0], args[1]
	log.Debug().Msgf("%s check %s %s", binaryName, remote, url)

	handleIAPAuthCookieFor(url)
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("%s %s\n", binaryName, version)
}

func installGitProtocol(cmd *cobra.Command, args []string) {
	p := strings.TrimLeft(binaryName, "git-remote-")
	git.InstallProtocol(p)
	log.Info().Msgf("%s protocol configured in git!", p)
}

func configureIAP(cmd *cobra.Command, args []string) {
	repo, err := _url.Parse(repoURL)
	https := fmt.Sprintf("https://%s", repo.Host)
	if err != nil {
		log.Error().Msgf("Could not convert %s in https://: %s", https, err)
	}

	log.Info().Msgf("Configure IAP for %s", https)
	git.SetGlobalConfig(https, "iap", "helperID", helperID)
	git.SetGlobalConfig(https, "iap", "helperSecret", helperSecret)
	git.SetGlobalConfig(https, "iap", "clientID", clientID)

	// let users manipulate standard 'https://' urls
	insteadOf := &git.GitConfig{
		Url:     fmt.Sprintf("https+iap://%s", repo.Host),
		Section: "url",
		Key:     "insteadOf",
		Value:   https,
	}
	if strings.Contains(repo.Host, "*") {
		log.Warn().Msg("While config is valid for wildcard hosts, transparent support for https:// remotes require \"insteadOf\" config")
		log.Info().Msg("Actual hosts must be manually configured as follows (with * replaced by subdomain):")
		log.Info().Msg(insteadOf.CommandSuggestGlobal())
	} else {
		git.SetConfigGlobal(insteadOf)
	}

	// set cookie path
	domainSlug := strings.ReplaceAll(repo.Host, ".", "-")
	domainSlug = strings.ReplaceAll(domainSlug, "*", "_wildcard_")
	cookiePath := fmt.Sprintf("~/.config/gcp-iap/%s.cookie", domainSlug)
	git.SetGlobalConfig(https, "http", "cookieFile", cookiePath)
}

func handleIAPAuthCookieFor(url string) *iap.Cookie {
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

	return cookie
}

func toHTTPSBaseDomain(addr string) (string, error) {
	u, err := _url.Parse(addr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s", u.Host), nil
}
