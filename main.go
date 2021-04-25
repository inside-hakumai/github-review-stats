package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
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
	ReviewPercentage float64 `json:"reviewPercentage"`
	ReviewedPRs int `json:"reviewedPRs"`
	PRsCreatedByOthers int `json:"PRsCreatedByOthers"`
	AllPRs int `json:"allPRs"`
}


func aggregateRepositoryReview(query GhQuery, user string) (percentage float64, reviewed int, target int, total int) {
	reviewed = 0
	target = 0
	total = 0

	for _, pr := range query.Repository.PullRequests.Nodes {
		total += 1

		// プルリクエストのauthorが自分ではない場合（レビュー率集計の対象である場合）
		if string(pr.Author.Login) != user {
			target += 1
			isReviewed := false

			for _, timelineItem := range pr.TimelineItems.Nodes {
				if user == string(timelineItem.PullRequestReview.Author.Login) ||
					user == string(timelineItem.IssueComment.Author.Login) ||
					user == string(timelineItem.ClosedEvent.Actor.Login) {
					isReviewed = true
				}

				if isReviewed == true {
					break
				}
			}

			if isReviewed {
				reviewed += 1
			}
		}
	}

	percentage = float64(reviewed) / float64(target)
	return
}

func main() {
	githubHost := os.Getenv("GHRS_HOST")
	if len(githubHost) == 0 {
		githubHost = "api.github.com"
	}

	targetRepos, err := getTargetRepositoryFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	accessToken := os.Getenv("GHRS_ACCESS_TOKEN")

	loginUser, err := askWhoAmI(githubHost, accessToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Logined to %s as %s\n", githubHost, *loginUser)

	for _, targetRepo := range targetRepos {
		//var url string
		slice := strings.Split(targetRepo, "/")
		if len(slice) != 2 {
			log.Fatal("Invalid repository format")
		}

		query, err := executeGhQueryForRepository(githubHost, slice[0], slice[1], accessToken)
		if err != nil {
			log.Fatal(err)
		}

		percentage, reviewed, target, total := aggregateRepositoryReview(*query, *loginUser)

		outputJsonData := OutputJson{
			ReviewPercentage:   percentage,
			ReviewedPRs:        reviewed,
			PRsCreatedByOthers: target,
			AllPRs:             total,
		}
		outputJsonText, err := json.Marshal(&outputJsonData)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s", outputJsonText)

		//if githubHost == "api.github.com" {
		//	url = fmt.Sprintf("https://%s/repos/%s/pulls", githubHost, targetRepo)
		//} else {
		//	url = fmt.Sprintf("https://%s/api/v3/repos/%s/pulls", githubHost, targetRepo)
		//}
		//
		//req, err := http.NewRequest("GET", url, nil)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//if len(accessToken) != 0 {
		//	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
		//}
		//
		//client := new(http.Client)
		//res, err := client.Do(req)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//defer res.Body.Close()
		//
		//if res.StatusCode == 404 {
		//	log.Fatalf("GitHub api returns 404. Original message: \n"+
		//		"Request url: %s\n"+
		//		"Original message: %s",
		//		url, res.Status)
		//}
		//
		//byteArray, _ := ioutil.ReadAll(res.Body)
		//
		//var response []*PullRequestData
		//err = json.Unmarshal(byteArray, &response)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//
		//fmt.Println(response)
		//appDirPath, err := ensureAppDirPath()
		//if err != nil {
		//	log.Fatal(err)
		//}
		//print(appDirPath)
	}
}

func getTargetRepositoryFromEnv() ([]string, error) {
	targetRepos := strings.Split(os.Getenv("GHRS_TARGET_REPOS"), ",")
	if len(targetRepos) == 1 && targetRepos[0] == "" {
		return nil, errors.New("Failed to get target repository from environment variable")
	}
	return targetRepos, nil
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

func executeGhQueryForRepository(githubHost string, repositoryOwner string, repositoryName string, accessToken string) (*GhQuery, error) {

	// TODO: githubHostの末尾に / があるかどうかを考慮する
	graphqlEndpoint := fmt.Sprintf("https://%s/graphql", githubHost)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewEnterpriseClient(graphqlEndpoint, httpClient)

	query := GhQuery{}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repositoryOwner),
		"repositoryName":  githubv4.String(repositoryName),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// _, _ = pp.Println(query)

	return &query, nil
}
