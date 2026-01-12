package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ot "gofast/pkg/otel"
	"gofast/service-core/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

type Provider string

const (
	ProviderFacebook  Provider = "facebook"
	ProviderGithub    Provider = "github"
	ProviderGoogle    Provider = "google"
	ProviderMicrosoft Provider = "microsoft"
)

type Info struct {
	Sub    string
	Email  string
	Avatar string
}

type OAuthClient interface {
	AuthCodeURL(cfg *config.Config, provider Provider, state string, opts ...oauth2.AuthCodeOption) (string, error)
	Exchange(ctx context.Context, cfg *config.Config, provider Provider, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, provider Provider, accessToken string) (*Info, error)
}

type OAuth struct{}

func getOAuthConfig(cfg *config.Config, provider Provider) (*oauth2.Config, error) {
	switch provider {
	case ProviderGoogle:
		return &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  cfg.OAuthRedirectURI,
			Scopes:       []string{"profile", "email", "openid"},
		}, nil
	case ProviderGithub:
		return &oauth2.Config{
			ClientID:     cfg.GithubClientID,
			ClientSecret: cfg.GithubClientSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  cfg.OAuthRedirectURI,
			Scopes:       []string{"user:email"},
		}, nil
	case ProviderFacebook:
		return &oauth2.Config{
			ClientID:     cfg.FacebookClientID,
			ClientSecret: cfg.FacebookClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.facebook.com/v10.0/dialog/oauth",
				TokenURL: "https://graph.facebook.com/v10.0/oauth/access_token",
			},
			RedirectURL: cfg.OAuthRedirectURI,
			Scopes:      []string{"email"},
		}, nil
	case ProviderMicrosoft:
		return &oauth2.Config{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
			Endpoint:     microsoft.AzureADEndpoint("common"),
			RedirectURL:  cfg.OAuthRedirectURI,
			Scopes:       []string{"profile", "email", "openid", "User.Read"},
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

func (OAuth) AuthCodeURL(cfg *config.Config, provider Provider, state string, opts ...oauth2.AuthCodeOption) (string, error) {
	oauthCfg, err := getOAuthConfig(cfg, provider)
	if err != nil {
		return "", err
	}
	return oauthCfg.AuthCodeURL(state, opts...), nil
}

func (OAuth) Exchange(ctx context.Context, cfg *config.Config, provider Provider, code string, opts ...oauth2.AuthCodeOption) (result *oauth2.Token, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.oauth.Exchange")
	defer func() { done(err) }()

	oauthCfg, err := getOAuthConfig(cfg, provider)
	if err != nil {
		return nil, err
	}
	token, err := oauthCfg.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}
	span.AddEvent("OAuth token exchanged")
	return token, nil
}

func (OAuth) GetUserInfo(ctx context.Context, provider Provider, accessToken string) (result *Info, err error) {
	ctx, _, done := ot.StartSpan(ctx, "login.oauth.GetUserInfo")
	defer func() { done(err) }()

	switch provider {
	case ProviderGoogle:
		return getGoogleUserInfo(ctx, accessToken)
	case ProviderGithub:
		return getGithubUserInfo(ctx, accessToken)
	case ProviderFacebook:
		return getFacebookUserInfo(ctx, accessToken)
	case ProviderMicrosoft:
		return getMicrosoftUserInfo(ctx, accessToken)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

func getGoogleUserInfo(ctx context.Context, accessToken string) (result *Info, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.oauth.getGoogleUserInfo")
	defer func() { done(err) }()

	body, err := HTTPCall(ctx, "https://www.googleapis.com/oauth2/v2/userinfo", accessToken)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	var data map[string]any
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	sub, ok := data["id"].(string)
	if !ok {
		return nil, errors.New("invalid user id")
	}
	email, _ := data["email"].(string)
	avatar, _ := data["picture"].(string)
	span.AddEvent("Google user info fetched")
	return &Info{Sub: sub, Email: email, Avatar: avatar}, nil
}

func getGithubUserInfo(ctx context.Context, accessToken string) (result *Info, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.oauth.getGithubUserInfo")
	defer func() { done(err) }()

	body, err := HTTPCall(ctx, "https://api.github.com/user", accessToken)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	var data map[string]any
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	userID, ok := data["id"].(float64)
	if !ok {
		return nil, errors.New("invalid user id")
	}
	sub := fmt.Sprintf("%.0f", userID)
	email, _ := data["email"].(string)
	avatar, _ := data["avatar_url"].(string)

	if email == "" {
		email, err = getGithubPrimaryEmail(ctx, accessToken)
		if err != nil {
			return nil, err
		}
	}
	span.AddEvent("GitHub user info fetched")
	return &Info{Sub: sub, Email: email, Avatar: avatar}, nil
}

func getGithubPrimaryEmail(ctx context.Context, accessToken string) (result string, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.oauth.getGithubPrimaryEmail")
	defer func() { done(err) }()

	body, err := HTTPCall(ctx, "https://api.github.com/user/emails", accessToken)
	if err != nil {
		return "", fmt.Errorf("http call: %w", err)
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	err = json.Unmarshal(body, &emails)
	if err != nil {
		return "", fmt.Errorf("json unmarshal: %w", err)
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			span.AddEvent("GitHub primary email found")
			return e.Email, nil
		}
	}
	return "", errors.New("no verified email found")
}

func getFacebookUserInfo(ctx context.Context, accessToken string) (result *Info, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.oauth.getFacebookUserInfo")
	defer func() { done(err) }()

	url := "https://graph.facebook.com/me?fields=id,name,email,picture&access_token=" + accessToken
	body, err := HTTPCall(ctx, url, accessToken)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	var data map[string]any
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	sub, ok := data["id"].(string)
	if !ok {
		return nil, errors.New("invalid user id")
	}
	email, _ := data["email"].(string)
	avatar := ""
	if pictureData, ok := data["picture"].(map[string]any); ok {
		if d, ok := pictureData["data"].(map[string]any); ok {
			avatar, _ = d["url"].(string)
		}
	}
	span.AddEvent("Facebook user info fetched")
	return &Info{Sub: sub, Email: email, Avatar: avatar}, nil
}

func getMicrosoftUserInfo(ctx context.Context, accessToken string) (result *Info, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.oauth.getMicrosoftUserInfo")
	defer func() { done(err) }()

	body, err := HTTPCall(ctx, "https://graph.microsoft.com/v1.0/me", accessToken)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	var data map[string]any
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	sub, ok := data["id"].(string)
	if !ok {
		return nil, errors.New("invalid user id")
	}
	email, _ := data["mail"].(string)
	if email == "" {
		email, _ = data["userPrincipalName"].(string)
	}
	avatar, _ := data["profilePhotoUrl"].(string)
	span.AddEvent("Microsoft user info fetched")
	return &Info{Sub: sub, Email: email, Avatar: avatar}, nil
}
