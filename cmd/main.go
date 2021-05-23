package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
)

func Exec(values FlagValues) error {

	/**
	ログインを試行
	*/
	loginUser, err := askWhoAmI(values.githubHostDomain, values.token)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Printf("Logined to %s as %s\n", values.githubHostDomain, *loginUser)

	/**
	プルリクエストを取得してレビュー率を算出
	*/
	stats, err := CalcReviewPercentageOverall(values.repositories, values.githubHostDomain, values.token, values.userId)
	if err != nil {
		return errors.WithStack(err)
	}

	/**
	算出したレビュー率を出力
	*/
	err = PrintToStdout(*stats)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
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
