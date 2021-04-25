package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
)

type User struct {
	Login githubv4.String
}

type PageInfo struct {
	HasNextPage githubv4.Boolean
	EndCursor   githubv4.String
}

type GhQuery struct {
	Repository struct {
		Name         githubv4.String
		PullRequests struct {
			PageInfo   PageInfo
			TotalCount githubv4.Int
			Nodes      []struct {
				Author        User `graphql:"author"`
				Title         githubv4.String
				TimelineItems struct {
					PageInfo   PageInfo
					TotalCount githubv4.Int
					Nodes      []struct {
						TypeName          githubv4.String `graphql:"__typename"`
						PullRequestReview struct {
							Author User `graphql:"author"`
						} `graphql:"... on PullRequestReview"`
						IssueComment struct {
							Author User `graphql:"author"`
						} `graphql:"... on IssueComment"`
						ClosedEvent struct {
							Actor User `graphql:"actor"`
						} `graphql:"... on ClosedEvent"`
					}
				} `graphql:"timelineItems(first: 100)"`
			}
		} `graphql:"pullRequests(first: 100)"`
	} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
}

type PullRequest struct {
	Id            int
	Title         string
	isContributed bool
}

type GhRepositoryQuerySimplified struct {
	RepoName     string
	PullRequests []PullRequest
}

type OutputJson struct {
	ReviewPercentage   float64 `json:"reviewPercentage"`
	ReviewedPRs        int     `json:"reviewedPRs"`
	PRsCreatedByOthers int     `json:"PRsCreatedByOthers"`
	AllPRs             int     `json:"allPRs"`
}

func Exec() error {
	githubHost := GetGitHubHostDomain()

	targetRepos, err := GetInspectionTargetRepository()
	if err != nil {
		return errors.WithStack(err)
	}

	accessToken := GetGitHubAccessToken()

	loginUser, err := askWhoAmI(githubHost, accessToken)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Printf("Logined to %s as %s\n", githubHost, *loginUser)

	for _, targetRepo := range targetRepos {
		percentage, reviewed, target, total, err := CalcReviewPercentageForSingleRepo(targetRepo, githubHost, accessToken, loginUser)
		if err != nil {
			return errors.WithStack(err)
		}

		outputJsonData := OutputJson{
			ReviewPercentage:   percentage,
			ReviewedPRs:        reviewed,
			PRsCreatedByOthers: target,
			AllPRs:             total,
		}
		outputJsonText, err := json.Marshal(&outputJsonData)
		if err != nil {
			return errors.WithStack(err)
		}

		fmt.Printf("%s", outputJsonText)
	}

	return nil
}

func ensureAppDirPath() (*string, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	// TODO: macOS以外のOSに対応
	appDirPath := filepath.FromSlash(usr.HomeDir + "/Library/Application Support/dev.insidehakumai.ghreviewstats")

	if _, err = os.Stat(appDirPath); os.IsNotExist(err) {
		// Create the new config file.
		if err := os.MkdirAll(appDirPath, 0755); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return &appDirPath, nil
}

func askWhoAmI(githubHost string, accessToken string) (*string, error) {
	var url string
	if githubHost == "api.github.com" {
		url = fmt.Sprintf("https://%s/user", githubHost)
	} else {
		url = fmt.Sprintf("https://%s/api/v3/user", githubHost)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(accessToken) != 0 {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	}

	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("GitHub api returns %d.\n"+
			"Request url: %s\n"+
			"Original message: %s",
			res.StatusCode, url, res.Status))
	}

	byteArray, _ := ioutil.ReadAll(res.Body)

	var response struct {
		Login string `json:"login"`
	}
	err = json.Unmarshal(byteArray, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &response.Login, nil
}

