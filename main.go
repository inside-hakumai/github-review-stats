package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type PullRequestData struct {
	Id        int    `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

func main() {
	githubHost := os.Getenv("GHRS_HOST")
	if len(githubHost) == 0 {
		githubHost = "api.github.com"
	}
	targetRepos := strings.Split(os.Getenv("GHRS_TARGET_REPOS"), ",")
	if len(targetRepos) == 0 {
		log.Fatal("Failed to get target repository from environment variable")
	}
	accessToken := os.Getenv("GHRS_ACCESS_TOKEN")

	for _, targetRepo := range targetRepos {
		var url string
		if githubHost == "api.github.com" {
			url = fmt.Sprintf("https://%s/repos/%s/pulls", githubHost, targetRepo)
		} else {
			url = fmt.Sprintf("https://%s/api/v3/repos/%s/pulls", githubHost, targetRepo)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}
		if len(accessToken) != 0 {
			req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
		}

		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		if res.StatusCode == 404 {
			log.Fatalf("GitHub api returns 404. Original message: \n"+
				"Request url: %s\n"+
				"Original message: %s",
				url, res.Status)
		}

		byteArray, _ := ioutil.ReadAll(res.Body)

		var response []*PullRequestData
		err = json.Unmarshal(byteArray, &response)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(response)
		appDirPath, err := ensureAppDirPath()
		if err != nil {
			log.Fatal(err)
		}
		print(appDirPath)
	}
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
			return nil, err
		}
	}

	return &appDirPath, nil
}
