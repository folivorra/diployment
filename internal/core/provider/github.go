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

const (
	githubUserInfoURL      = "https://api.github.com/user"
	githubListUserReposURL = "https://api.github.com/user/repos?per_page=100"

	githubAcceptContent = "application/vnd.github+json"
	githubAPIVersion    = "2026-03-10"
)

type githubUser struct {
	ID        int    `json:"id"`
	AvatarURL string `json:"avatar_url"`
}

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

	resp, err := client.Get(githubUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("send get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user from GitHub API: status %s", resp.Status)
	}

	var ghUser githubUser
	if err = json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}

	return &model.User{
		GithubID:  ghUser.ID,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}

func (g *GitHubProvider) ListUserRepos(ctx context.Context, ownerToken string) ([]*model.Repository, error) {
	headers := &http.Header{
		"Accept":               []string{githubAcceptContent},
		"Authorization":        []string{"Bearer " + ownerToken},
		"X-GitHub-Api-Version": []string{githubAPIVersion},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubListUserReposURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header = *headers

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("do request: status %s", resp.Status)
	}

	var repos []*model.Repository
	if err = json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}

	return repos, nil
}
