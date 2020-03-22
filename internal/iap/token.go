package iap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/adohkan/git-remote-https-iap/internal/git"
	"github.com/int128/oauth2cli"
	"github.com/pkg/browser"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/errgroup"
)

const (
	CacheProtocol = "iap"
	CacheUsername = "refresh-token"
)

type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	IDToken     string `json:"id_token"`
}

// getRefreshTokenFromBrowserFlow initialize an OAuth login workflow via the browser and returns a refresh token valid for a given url
// see: https://github.com/int128/oauth2cli/blob/master/example/main.go
func getRefreshTokenFromBrowserFlow(domain, helperID, helperSecret string) (string, error) {
	ctx := context.Background()
	ready := make(chan string, 1)

	var eg errgroup.Group
	var token *oauth2.Token
	var err error

	var OAuthConfig = oauth2.Config{
		ClientID:     helperID,
		ClientSecret: helperSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"openid", "email"},
	}

	eg.Go(func() error {
		select {
		case url, ok := <-ready:
			if !ok {
				return nil
			}
			log.Debug().Msgf("getRefreshTokenFromBrowserFlow - Open %s", url)
			if err := browser.OpenURL(url); err != nil {
				log.Error().Msgf("getRefreshTokenFromBrowserFlow - could not open the browser: %s", err)
			}
			return nil
		case err := <-ctx.Done():
			return fmt.Errorf("context done while waiting for authorization: %w", err)
		}
	})

	eg.Go(func() error {
		defer close(ready)

		cfg := oauth2cli.Config{
			OAuth2Config:         OAuthConfig,
			LocalServerReadyChan: ready,
		}

		token, err = oauth2cli.GetToken(ctx, cfg)
		if err != nil {
			return fmt.Errorf("could not get 'access_token' for the desktop-app: %w", err)
		}

		return nil
	})

	err = eg.Wait()
	if err != nil {
		return "", err
	}

	log.Debug().Msgf("getRefreshTokenFromBrowserFlow - refresToken: %s", token.RefreshToken)
	return token.RefreshToken, nil
}

func cacheRefreshToken(key, token string) error {
	return git.StoreCredentials(CacheProtocol, key, CacheUsername, token)
}

func getRefreshTokenFromCache(key string) (string, error) {
	return git.GetCredentials(CacheProtocol, key, CacheUsername)
}

// GetIAPAuthToken returns a raw IAP auth token for the given args
func GetIAPAuthToken(domain, helperID, helperSecret, IAPclientID string) (string, error) {
	var result Token

	refreshToken, err := getRefreshTokenFromCache(domain)
	if err != nil {
		log.Debug().Msgf("no cached refresh token for %s: %s", domain, err.Error())

		refreshToken, err = getRefreshTokenFromBrowserFlow(domain, helperID, helperSecret)
		if err != nil {
			return "", err
		}
		if err := cacheRefreshToken(domain, refreshToken); err != nil {
			log.Warn().Msgf("could not cache refresh token for %s: %s", domain, err.Error())
		}
	}

	// exchange our refreshToken for an id_token that we can use as GCP_IAAP_AUTH_TOKEN
	resp, err := http.PostForm(google.Endpoint.TokenURL, url.Values{
		"client_id":     {helperID},
		"client_secret": {helperSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"audience":      {IAPclientID},
	})

	if err != nil {
		return "", fmt.Errorf("could not get exchange 'refresh_token' for IAP Auth Token: %s", err.Error())
	}

	log.Debug().Msgf("GetIAPAuthToken - successfully used 'refresh_token' to claim IAP Auth Token")

	json.NewDecoder(resp.Body).Decode(&result)

	return result.IDToken, nil
}
