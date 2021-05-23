package cmd

import (
	"github.com/spf13/cobra"
)

type FlagValues struct {
	userId           string
	githubHostDomain string
	repositories     []string
	token            string
}

var (
	userId           string
	githubHostDomain string
	repositories     []string
	token            string

	rootCmd = &cobra.Command{
		Use:   "github-review-stats",
		Short: "hoge",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}
)

func initCommand() {
	rootCmd.Flags().StringVarP(&githubHostDomain, "ghHostDomain", "d", "api.github.com", "GitHub APIのドメイン")
	rootCmd.Flags().StringVarP(&userId, "userId", "u", "", "集計対象のGitHubアカウントID（必須）")
	rootCmd.Flags().StringSliceVarP(&repositories, "repositories", "r", nil, "集計対象のリポジトリ（カンマ区切りもしくは複数回指定、必須）")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "GitHubのアクセストークン")

	rootCmd.MarkFlagRequired("userId")
	rootCmd.MarkFlagRequired("repositories")
	rootCmd.MarkFlagRequired("token") // TODO: パスワードログインも可能にした時にこの行を削除する
}

func ParseCommandLineArguments() (*FlagValues, bool) {
	initCommand()

	err := rootCmd.Execute()
	if err != nil {
		return nil, false
	}

	return &FlagValues{
		userId:           userId,
		githubHostDomain: githubHostDomain,
		repositories:     repositories,
		token:            token,
	}, true
}
