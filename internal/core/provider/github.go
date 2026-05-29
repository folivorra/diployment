package provider

import (
	"bytes"
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
	githubListBranchesURL  = "https://api.github.com/repos/%s/branches"
	githubCreateWebhookURL = "https://api.github.com/repos/%s/hooks"
	githubDeleteWebhookURL = "https://api.github.com/repos/%s/hooks/%d"

	githubAcceptContent = "application/vnd.github+json"
	githubAPIVersion    = "2026-03-10"
)

type githubBranch struct {
	Name string `json:"name"`
}

type githubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type gitHubProvider struct {
	oauthCfg   *oauth2.Config
	webhookURL string
}

func NewGitHubProvider(ghCfg config.GitHubConfig, webhookURL string) *gitHubProvider {
	return &gitHubProvider{
		oauthCfg: &oauth2.Config{
			ClientID:     ghCfg.ClientID,
			ClientSecret: ghCfg.ClientSecret,
			RedirectURL:  ghCfg.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"repo", "read:user"},
		},
		webhookURL: webhookURL,
	}
}

func (g *gitHubProvider) AuthCodeURL(state string) string {
	return g.oauthCfg.AuthCodeURL(state)
}

func (g *gitHubProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.oauthCfg.Exchange(ctx, code)
}

func (g *gitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error) {
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
		Login:     ghUser.Login,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}

func (g *gitHubProvider) ListUserRepos(ctx context.Context, token string) ([]*model.Repository, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubListUserReposURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header = getHeaders(token)

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

func (g *gitHubProvider) ListRepoBranches(ctx context.Context, token string, repoFullName string) ([]string, error) {
	url := fmt.Sprintf(githubListBranchesURL, repoFullName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header = getHeaders(token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("do request: status %s", resp.Status)
	}

	var raw []githubBranch
	if err = json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}

	names := make([]string, len(raw))
	for i, b := range raw {
		names[i] = b.Name
	}
	return names, nil
}

func (g *gitHubProvider) CreateWebhook(ctx context.Context, token string, repoFullName string, webhookSecret string) (int, error) {
	params := struct {
		Name   string            `json:"name"`
		Active bool              `json:"active"`
		Events []string          `json:"events"`
		Config map[string]string `json:"config"`
	}{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: map[string]string{
			"url":          g.webhookURL,
			"content_type": "json",
			"secret":       webhookSecret,
		},
	}

	reqBody, err := json.Marshal(params)
	if err != nil {
		return -1, fmt.Errorf("marshal request body: %w", err)
	}

	url := fmt.Sprintf(githubCreateWebhookURL, repoFullName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return -1, fmt.Errorf("create request: %w", err)
	}
	req.Header = getHeaders(token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return -1, fmt.Errorf("do request: status %s", resp.Status)
	}

	var webhookID struct {
		ID int `json:"id"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&webhookID); err != nil {
		return -1, fmt.Errorf("decode response body: %w", err)
	}

	return webhookID.ID, nil
}

func (g *gitHubProvider) DeleteWebhook(ctx context.Context, token string, repoFullName string, webhookID int) error {
	url := fmt.Sprintf(githubDeleteWebhookURL, repoFullName, webhookID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header = getHeaders(token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("do request: status %s", resp.Status)
	}

	return nil
}

func getHeaders(token string) http.Header {
	return http.Header{
		"Accept":               []string{githubAcceptContent},
		"Authorization":        []string{"Bearer " + token},
		"X-GitHub-Api-Version": []string{githubAPIVersion},
	}
}
