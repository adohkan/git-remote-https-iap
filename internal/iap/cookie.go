package iap

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/adohkan/git-remote-https-iap/internal/git"
	"github.com/rs/zerolog/log"
)

const (
	// IAPCookieName is the name of the HTTP Cookie that will be used to send the IAP Token.
	// see: https://cloud.google.com/blog/products/gcp/getting-started-with-cloud-identity-aware-proxy
	IAPCookieName = "GCP_IAAP_AUTH_TOKEN"
)

type Cookie struct {
	JarPath string
	Domain  string
	Token   jwt.Token
	Claims  jwt.StandardClaims
}

// ReadCookie lookup the http.cookieFile for a given domain and try to load it from the filesystem
func ReadCookie(domain string) (*Cookie, error) {
	cookieFile := git.ConfigGetURLMatch("http.cookieFile", domain)

	url, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	c := Cookie{
		JarPath: cookieFile,
		Domain:  url.Host,
	}

	rawToken, err := c.readRawTokenFromJar()
	if err != nil {
		return nil, err
	}

	token, claims, err := parseJWToken(rawToken)
	if err != nil {
		return nil, err
	}

	c.Token = token
	c.Claims = claims

	return &c, nil
}

func (c *Cookie) readRawTokenFromJar() (string, error) {
	path := expandHome(c.JarPath)

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			log.Warn().Msgf("readRawTokenFromJar - unexpected format while parsing IAP cookie: %v", line)
			continue
		}
		// see: https://curl.haxx.se/docs/http-cookies.html
		cookieName, cookieValue := fields[5], strings.TrimSpace(fields[6])
		if cookieName != IAPCookieName {
			log.Debug().Msgf("readRawTokenFromJar - skip '%s' while parsing IAP cookie", cookieName)
			continue
		}

		return cookieValue, nil
	}
	return "", fmt.Errorf("readRawTokenFromJar - %s not found", IAPCookieName)
}

// NewCookie takes care of the authentication workflow and creates the relevant IAP Cookie on the filesystem
func NewCookie(domain string) (*Cookie, error) {

	helperID := git.ConfigGetURLMatch("iap.helperID", domain)
	helperSecret := git.ConfigGetURLMatch("iap.helperSecret", domain)
	IAPClientID := git.ConfigGetURLMatch("iap.clientID", domain)
	cookieFile := git.ConfigGetURLMatch("http.cookieFile", domain)

	url, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	rawToken, err := GetIAPAuthToken(domain, helperID, helperSecret, IAPClientID)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("rawToken: %+v", rawToken)

	token, claims, err := parseJWToken(rawToken)
	if err != nil {
		return nil, err
	}

	c := Cookie{
		JarPath: cookieFile,
		Domain:  url.Host,
		Token:   token,
		Claims:  claims,
	}
	return &c, c.write(token.Raw, claims.ExpiresAt)
}

func (c *Cookie) write(token string, exp int64) error {
	path := expandHome(c.JarPath)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}

	if err := os.Chmod(path, 0600); err != nil {
		return err
	}

	if _, err = f.WriteString(fmt.Sprintf("%s\tx\tx\tx\t%d\t%s\t%s\n", c.Domain, exp, IAPCookieName, token)); err != nil {
		return err
	}

	return nil
}

// Expired returns a boolean that indicate if the expires-at claim is in the future
func (c *Cookie) Expired() bool {
	return c.Claims.ExpiresAt < time.Now().Unix()
}

func parseJWToken(rawToken string) (jwt.Token, jwt.StandardClaims, error) {
	var p jwt.Parser
	var claims jwt.StandardClaims

	token, _, err := p.ParseUnverified(rawToken, &claims)
	return *token, claims, err
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}

	return filepath.Join(os.Getenv("HOME"), path[1:])
}
