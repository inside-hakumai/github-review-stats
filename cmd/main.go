package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
)

func Exec() error {

	/**
	GitHubに接続するための各種設定値を取得
	*/
	githubHost := GetGitHubHostDomain()

	targetRepos, err := GetInspectionTargetRepository()
	if err != nil {
		return errors.WithStack(err)
	}

	accessToken := GetGitHubAccessToken()

	/**
	ログインを試行
	*/
	loginUser, err := askWhoAmI(githubHost, accessToken)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Printf("Logined to %s as %s\n", githubHost, *loginUser)

	/**
	プルリクエストを取得してレビュー率を算出
	*/
	stats, err := CalcReviewPercentageOverall(targetRepos, githubHost, accessToken, *loginUser)
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
