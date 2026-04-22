package model

type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Private  bool   `json:"private"`
	CloneURL string `json:"clone_url"`
}
