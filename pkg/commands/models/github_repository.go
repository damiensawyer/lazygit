package models

import "time"

// GithubRepository is a repository on GitHub, as listed when looking for one
// to clone.
type GithubRepository struct {
	// e.g. "jesseduffield/lazygit"
	FullName    string
	Owner       string
	Name        string
	Description string
	IsFork      bool
	IsPrivate   bool
	IsArchived  bool
	Stars       int
	Forks       int
	// The repository's primary language, or empty if GitHub doesn't know it.
	Language string
	PushedAt time.Time
	// The web page, e.g. "https://github.com/jesseduffield/lazygit"
	HTMLURL  string
	SSHURL   string
	CloneURL string
	// The repository this one is a fork of; nil if it isn't a fork, or if the
	// listing didn't say.
	Parent *GithubRepository
}
