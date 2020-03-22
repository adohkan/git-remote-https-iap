package iap

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

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
