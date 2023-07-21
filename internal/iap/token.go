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
	// CacheProtocol is the protocol used when saving the refresh-token in git-credential-store
	// It can be an arbitrary value.
	CacheProtocol = "iap"

	// CacheUsername is the username used when saving the refresh-token in git-credential-store.
	// It can be an arbitrary value.
	CacheUsername = "refresh-token"
)

type token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	IDToken     string `json:"id_token"`
}

type httpError struct {
	Error     string `json:"error"`
	ErrorDesc string `json:"error_description"`
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
			log.Debug().Msgf("[getRefreshTokenFromBrowserFlow] Open %s", url)
			if err := browser.OpenURL(url); err != nil {
				log.Error().Msgf("[getRefreshTokenFromBrowserFlow] Could not open the browser: %s", err)
			}
			return nil
		case err := <-ctx.Done():
			return fmt.Errorf("[getRefreshTokenFromBrowserFlow] Context done while waiting for authorization: %w", err)
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
			return fmt.Errorf("[getRefreshTokenFromBrowserFlow] Could not get 'access_token' for the desktop-app: %w", err)
		}

		return nil
	})

	err = eg.Wait()
	if err != nil {
		return "", err
	}

	log.Debug().Msgf("[getRefreshTokenFromBrowserFlow] refreshToken: %s", token.RefreshToken)
	return token.RefreshToken, nil
}

func cacheRefreshToken(key, token string) error {
	return git.StoreCredentials(CacheProtocol, key, CacheUsername, token)
}

func getRefreshTokenFromCache(key string) (string, error) {
	return git.GetCredentials(CacheProtocol, key, CacheUsername)
}

// GetIAPAuthToken take care of the IAP Authentication process when relevant.
// It optmize this workflow by detecting cases where an existing IAP auth token is already available,
// and caching a refresh-token.
// It returns a raw IAP auth token and any error encountered.
func GetIAPAuthToken(domain, helperID, helperSecret, IAPclientID string, forcebrowserflow bool) (string, error) {
	var result token
	var errorMesg httpError

	refreshToken, err := getRefreshTokenFromCache(domain)

	if forcebrowserflow {
		log.Debug().Msgf("[GetIAPAuthToken] Forcing getRefreshTokenFromBrowserFlow")
		refreshToken, err = getRefreshTokenFromBrowserFlow(domain, helperID, helperSecret)
	}

	if err != nil {
		log.Debug().Msgf("[GetIAPAuthToken] No cached refresh token for %s: %s", domain, err.Error())

		refreshToken, err = getRefreshTokenFromBrowserFlow(domain, helperID, helperSecret)
		if err != nil {
			log.Debug().Msgf("[GetIAPAuthToken] getRefreshTokenFromBrowserFlow Failed")
			return "", err
		}
		if err := cacheRefreshToken(domain, refreshToken); err != nil {
			log.Warn().Msgf("[GetIAPAuthToken] Could not cache refresh token for %s: %s", domain, err.Error())
		}
	}
	log.Debug().Msgf("[GetIAPAuthToken] refreshToken is: %s", refreshToken)

	// exchange our refreshToken for an id_token that we can use as GCP_IAAP_AUTH_TOKEN
	log.Debug().Msgf("[GetIAPAuthToken] Google Endpoint is: %s", google.Endpoint.TokenURL)
	resp, err := http.PostForm(google.Endpoint.TokenURL, url.Values{
		"client_id":     {helperID},
		"client_secret": {helperSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"audience":      {IAPclientID},
	})

	if err != nil {
		return "", fmt.Errorf("[GetIAPAuthToken] Could not get exchange 'refresh_token' for IAP Auth Token: %s", err.Error())
	}

	if resp.StatusCode != 200 {
		json.NewDecoder(resp.Body).Decode(&errorMesg)
		return "", fmt.Errorf("[GetIAPAuthToken] Could not get exchange 'refresh_token' for IAP Auth Token: HTTP Error Code: %s .... Error Description: %s", errorMesg.ErrorDesc, errorMesg.Error)
	}

	log.Debug().Msgf("[GetIAPAuthToken] Successfully used 'refresh_token' to claim IAP Auth Token")

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("[GetIAPAuthToken] Could not get exchange 'refresh_token' for IAP Auth Token: %s", err.Error())
	}

	return result.IDToken, nil
}
