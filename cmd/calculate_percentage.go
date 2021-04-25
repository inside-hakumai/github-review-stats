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
	Reviewed int
	Target int
	Total int
}

func CalcReviewPercentageForSingleRepo(targetRepo string, githubHost string, accessToken string, loginUser *string) (float64, int, int, int, error) {
	slice := strings.Split(targetRepo, "/")
	if len(slice) != 2 {
		return 0, 0, 0, 0, errors.New("Invalid repository format")
	}

	query, err := executeGhQueryForRepository(githubHost, slice[0], slice[1], accessToken)
	if err != nil {
		return 0, 0, 0, 0, errors.WithStack(err)
	}

	reviewed, createdByOthers, total := countPRsNumber(*query, *loginUser)
	return float64(reviewed) / float64(createdByOthers) , reviewed, createdByOthers, total, nil
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
