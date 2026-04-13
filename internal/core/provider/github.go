package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/folivorra/diployment/internal/config"
	"github.com/folivorra/diployment/internal/model"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubProvider struct {
	oauthCfg *oauth2.Config
}

func NewGitHubProvider(cfg config.GitHubConfig) *GitHubProvider {
	return &GitHubProvider{
		oauthCfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"repo", "read:user"},
		},
	}
}

func (g *GitHubProvider) AuthCodeURL(state string) string {
	return g.oauthCfg.AuthCodeURL(state)
}

func (g *GitHubProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.oauthCfg.Exchange(ctx, code)
}

func (g *GitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error) {
	client := g.oauthCfg.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("send get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user from GitHub API: %w", err)
	}

	var extUser *model.ExternalUser
	if err = json.NewDecoder(resp.Body).Decode(&extUser); err != nil {
		return nil, fmt.Errorf("decode response body to json: %w", err)
	}

	user := &model.User{
		GithubID:  extUser.ID,
		AvatarURL: extUser.AvatarURL,
	}

	return user, nil
}
