package cmd

import (
	"github.com/pkg/errors"
	"os"
	"strings"
)

func GetGitHubHostDomain() string {
	githubHost := os.Getenv("GHRS_HOST")
	if len(githubHost) == 0 {
		githubHost = "api.github.com"
	}

	return githubHost
}

func GetInspectionTargetRepository() ([]string, error) {
	targetRepos := strings.Split(os.Getenv("GHRS_TARGET_REPOS"), ",")
	if len(targetRepos) == 1 && targetRepos[0] == "" {
		return nil, errors.New("Failed to get target repository from environment variable")
	}
	return targetRepos, nil
}

func GetGitHubAccessToken() string {
	accessToken := os.Getenv("GHRS_ACCESS_TOKEN")

	return accessToken
}
