package cmd

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/trknhr/backlog-git-pr-diff-checker/defaults"
	"github.com/trknhr/gbch"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/trknhr/backlog-git-pr-diff-checker/exit"
	backlog "github.com/vvatanabe/go-backlog/backlog/v2"
)

var (
	since       string
	apiKey      string
	targetPaths []string
	description string
)

func createGbch(apiKey string, targetPaths []string) *gbch.Gbch {
	gb := &gbch.Gbch{
		APIKey:      apiKey,
		TargetPaths: targetPaths,
	}
	_ = gb.Initialize(context.Background())

	return gb
}

func RunRoot(cmd *cobra.Command, args []string) (string, error) {
	gbch := createGbch(apiKey, targetPaths)
	initViper()

	myDirHash := getMyDirHash()

	var s map[string]struct {
		LastCommit string `toml:"lastcommit"`
		Path       string `toml:"path"`
	}

	_ = viper.Unmarshal(&s)

	prevCommit := ""
	if val, ok := s[myDirHash]; ok {
		prevCommit = val.LastCommit
	}

	if prevCommit == "" && since == "" {
		since = "1 day ago"
	}

	if since != "" {
		out, err := exec.Command("sh", "-c", fmt.Sprintf("git log --since='%s' %s", since, "--pretty=format:%H | tail -n 1")).Output()
		if err != nil {
			return "", err
		}
		prevCommit = string(out)
	}

	targetPathsStr := []string{}
	for _, v := range targetPaths {
		targetPathsStr = append(targetPathsStr, fmt.Sprintf("--target-paths=%s", v))
	}

	section, err := gbch.GetSection(context.Background(), prevCommit, "")

	if err != nil {
		return "", err
	}

	outputSection := &OutputSection{
		Title:        "Diff PRs on this time",
		Description:  description,
		HTMLURL:      section.HTMLURL,
		FromRevision: section.FromRevision,
		ToRevision:   section.ToRevision,
		PullRequests: section.PullRequests,
		ChangedAt:    section.ChangedAt,
		BaseURL:      section.BaseURL,
		ShowUniqueID: section.ShowUniqueID,
	}

	if err != nil {
		return "", err
	}

	lastCommitOut, _ := exec.Command("sh", "-c", "git log --pretty=format:%H | head -n 1").Output()
	viper.Set(fmt.Sprintf("%s.lastCommit", myDirHash), strings.TrimSuffix(string(lastCommitOut), "\n"))

	mydir, err := os.Getwd()

	if err != nil {
		return "", err
	}
	viper.Set(fmt.Sprintf("%s.path", myDirHash), mydir)

	err = viper.WriteConfig()

	if err != nil {
		return "", err
	}
	if len(outputSection.PullRequests) > 0 {
		return display(outputSection)
	}

	return "There are no pull requests.", nil
}

func getMD5(str string) string {
	data := []byte(str)
	return fmt.Sprintf("%x", md5.Sum(data))
}

func getMyDirHash() string {
	mydir, err := os.Getwd()

	if err != nil {
		fmt.Println(err)
	}

	return getMD5(mydir)
}

func initViper() {
	home, err := homedir.Dir()

	cobra.CheckErr(err)

	configHome := home
	configName := ".backlog-git-pr-diff-checker"
	configType := "toml"

	viper.AddConfigPath(configHome)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	_ = viper.ReadInConfig()

	configPath := filepath.Join(configHome, configName+"."+configType)

	_, err = os.Stat(configPath)

	if !os.IsExist(err) {
		if _, err := os.Create(configPath); err != nil {
			cobra.CheckErr(err)
		}
	}
}

func runRootWrapper(cmd *cobra.Command, args []string) {
	if result, err := RunRoot(cmd, args); err != nil {
		exit.Fail(err)
	} else {
		exit.Succeed(result)
	}
}
func Execute() error {
	var rootCmd = &cobra.Command{
		Version: defaults.Version,
		Use:     "backlog-git-pr-diff-checker",
		Short:   "It checks Git for particular path",
		Run:     runRootWrapper,
	}
	rootCmd.PersistentFlags().StringSliceVarP(&targetPaths, "target-paths", "p", []string{}, "Target paths you want to filter.")
	rootCmd.PersistentFlags().StringVarP(&since, "since", "s", "", "Limit the commits to those made after the specified date.")
	rootCmd.PersistentFlags().StringVarP(&apiKey, "apikey", "k", "", "Backlog's api key")
	rootCmd.PersistentFlags().StringVarP(&description, "description", "d", "", "The name of this diff check")

	rootCmd.Use = ""

	return rootCmd.Execute()
}

type OutputSection struct {
	Description  string
	Title        string
	HTMLURL      string
	FromRevision string
	ToRevision   string
	PullRequests []*backlog.PullRequest
	ChangedAt    time.Time
	BaseURL      string
	ShowUniqueID bool
}

var markdownTmplStr = `{{$ret := . -}}
{{.Description}}
# [{{.Title}}]({{.HTMLURL}}/compare/{{.FromRevision}}...{{.ToRevision}}) ({{.ChangedAt.Format "2006-01-02"}})
{{range .PullRequests}}
* {{.Summary}} [#{{.Number}}]({{$ret.HTMLURL}}/pullRequests/{{.Number}}) ([{{.CreatedUser.Name}}]({{$ret.BaseURL}}/user/{{.CreatedUser.UserID}})){{if and ($ret.ShowUniqueID) (.CreatedUser.NulabAccount)}} @{{.CreatedUser.NulabAccount.UniqueID}}{{end}}
{{- end}}`

func display(rs *OutputSection) (string, error) {
	var b bytes.Buffer
	mdTmpl, _ := template.New("md-changelog").Parse(markdownTmplStr)

	err := mdTmpl.Execute(&b, rs)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
