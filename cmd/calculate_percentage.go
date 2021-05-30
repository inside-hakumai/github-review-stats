package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"strings"
)

type Stats struct {
	Percentage float64
	Reviewed   int
	Target     int
	Total      int
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

type User struct {
	Login githubv4.String
}

type PageInfo struct {
	HasNextPage githubv4.Boolean
	EndCursor   githubv4.String
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

type RepoReviewStats struct {
	RepoName           string
	ReviewPercentage   float64
	ReviewedPRs        int
	PRsCreatedByOthers int
	AllPRs             int
}

type OverallStats struct {
	ReviewPercentage   float64
	ReviewedPRs        int
	PRsCreatedByOthers int
	AllPRs             int
	RepoStatsList      []RepoReviewStats
}

func CalcReviewPercentageOverall(targetRepos []string, githubHost string, accessToken string, loginUser string) (*OverallStats, error) {
	numOfOverallReviewedPRs := 0
	numOfOverallPRsCreatedByOthers := 0
	numOfOverallTotalPRs := 0
	var repoStatsList []RepoReviewStats
	for _, targetRepo := range targetRepos {
		repoReviewStats, err := CalcReviewPercentageForSingleRepo(targetRepo, githubHost, accessToken, loginUser)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		repoStatsList = append(repoStatsList, *repoReviewStats)
		numOfOverallReviewedPRs += repoReviewStats.ReviewedPRs
		numOfOverallPRsCreatedByOthers += repoReviewStats.PRsCreatedByOthers
		numOfOverallTotalPRs += repoReviewStats.AllPRs
	}

	return &OverallStats{
		ReviewPercentage:   float64(numOfOverallReviewedPRs) / float64(numOfOverallPRsCreatedByOthers),
		ReviewedPRs:        numOfOverallReviewedPRs,
		PRsCreatedByOthers: numOfOverallPRsCreatedByOthers,
		AllPRs:             numOfOverallTotalPRs,
		RepoStatsList:      repoStatsList,
	}, nil

}

func CalcReviewPercentageForSingleRepo(targetRepo string, githubHost string, accessToken string, loginUser string) (*RepoReviewStats, error) {
	slice := strings.Split(targetRepo, "/")
	if len(slice) != 2 {
		return nil, errors.New("Invalid repository format")
	}

	query, err := executeGhQueryForRepository(githubHost, slice[0], slice[1], accessToken)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	reviewed, createdByOthers, total := countPRsNumber(*query, loginUser)
	return &RepoReviewStats{
		RepoName:           targetRepo,
		ReviewPercentage:   float64(reviewed) / float64(createdByOthers),
		ReviewedPRs:        reviewed,
		PRsCreatedByOthers: createdByOthers,
		AllPRs:             total,
	}, nil
}

func executeGhQueryForRepository(githubHost string, repositoryOwner string, repositoryName string, accessToken string) (*GhQuery, error) {

	var graphqlEndpoint string
	// TODO: githubHostの末尾に / があるかどうかを考慮する
	if githubHost == "api.github.com" {
		graphqlEndpoint = fmt.Sprintf("https://%s/graphql", githubHost)
	} else {
		graphqlEndpoint = fmt.Sprintf("https://%s/api/graphql", githubHost)
	}

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

func countPRsNumber(query GhQuery, user string) (reviewed int, createdByOthers int, total int) {
	reviewed = 0
	createdByOthers = 0
	total = 0

	for _, pr := range query.Repository.PullRequests.Nodes {
		total += 1

		// プルリクエストのauthorが自分ではない場合（レビュー率集計の対象である場合）
		if string(pr.Author.Login) != user {
			createdByOthers += 1
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

	return reviewed, createdByOthers, total
}
