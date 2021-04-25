package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
)

type overallJson struct {
	ReviewPercentage   float64 `json:"reviewPercentage"`
	ReviewedPRs        int     `json:"reviewedPRs"`
	PRsCreatedByOthers int     `json:"PRsCreatedByOthers"`
	AllPRs             int     `json:"allPRs"`
}

type byRepositoryJson struct {
	RepoName           string  `json:"repoName"`
	ReviewPercentage   float64 `json:"reviewPercentage"`
	ReviewedPRs        int     `json:"reviewedPRs"`
	PRsCreatedByOthers int     `json:"PRsCreatedByOthers"`
	AllPRs             int     `json:"allPRs"`
}

type OutputJson struct {
	Overall      overallJson        `json:"overall"`
	ByRepository []byRepositoryJson `json:"byRepository"`
}

func PrintToStdout(stats OverallStats) error {

	//goland:noinspection GoPreferNilSlice
	byRepositoryList := []byRepositoryJson{}

	for _, repoStats := range stats.RepoStatsList {
		byRepositoryList = append(byRepositoryList, byRepositoryJson{
			RepoName:           repoStats.RepoName,
			ReviewPercentage:   repoStats.ReviewPercentage,
			ReviewedPRs:        repoStats.ReviewedPRs,
			PRsCreatedByOthers: repoStats.PRsCreatedByOthers,
			AllPRs:             repoStats.AllPRs,
		})
	}

	outputJsonData := OutputJson{
		Overall: overallJson{
			ReviewPercentage:   stats.ReviewPercentage,
			ReviewedPRs:        stats.ReviewedPRs,
			PRsCreatedByOthers: stats.PRsCreatedByOthers,
			AllPRs:             stats.AllPRs,
		},
		ByRepository: byRepositoryList,
	}

	outputJsonText, err := json.MarshalIndent(&outputJsonData, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Printf("%s", outputJsonText)

	return nil
}
