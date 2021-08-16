package cmd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/trknhr/backlog-git-pr-diff-checker/backlog_pr"
	"github.com/trknhr/backlog-git-pr-diff-checker/defaults"
	"github.com/trknhr/backlog-git-pr-diff-checker/git_cmd"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/trknhr/backlog-git-pr-diff-checker/exit"
	backlog "github.com/vvatanabe/go-backlog/backlog/v2"
)

var (
	since          string
	apiKey         string
	targetPaths    []string
	description    string
	repoDir        string
	settingFileDir string
)

func RunRoot(cmd *cobra.Command, args []string) (string, error) {
	initViper()

	if repoDir == "" {
		mydir, err := os.Getwd()
		if err != nil {
			cobra.CheckErr(err)
		}
		repoDir = mydir
	}

	myDirHash := getMyDirHash(repoDir)

	var s map[string]struct {
		LastCommit string `toml:"lastcommit"`
		Path       string `toml:"path"`
	}

	_ = viper.Unmarshal(&s)

	prevCommit := ""
	if val, ok := s[myDirHash]; ok {
		prevCommit = val.LastCommit
	}

	gitCmd := git_cmd.NewGitCmd(repoDir)

	if prevCommit == "" || since != "" {
		if since == "" {
			since = "1 day ago"
		}
		output, err := gitCmd.Exec("log", "--reverse", fmt.Sprintf("--until='%s'", since), "--pretty=format:%H", "-1")
		if err != nil {
			cobra.CheckErr(err)
		}

		prevCommit = output
	}

	section, err := backlog_pr.GetPullRequest(&backlog_pr.BacklogGit{
		GitCmd:      gitCmd,
		LastCommit:  prevCommit,
		ApiKey:      apiKey,
		TargetPaths: targetPaths,
	})

	if err != nil {
		cobra.CheckErr(err)
	}

	outputSection := &OutputSection{
		Title:        "Diff PRs on this time",
		Description:  description,
		RepoURL:      section.RepoURL,
		PullRequests: section.PullRequests,
	}

	if err != nil {
		cobra.CheckErr(err)
	}

	lastCommitOut, _ := gitCmd.Exec("log", "-n", "1", "--pretty=format:%H")
	viper.Set(fmt.Sprintf("%s.lastCommit", myDirHash), strings.TrimSuffix(string(lastCommitOut), "\n"))

	viper.Set(fmt.Sprintf("%s.path", myDirHash), repoDir)

	err = viper.WriteConfig()

	if err != nil {
		cobra.CheckErr(err)
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

func getMyDirHash(dir string) string {
	return getMD5(dir)
}

func initViper() {
	if settingFileDir == "" {
		dirname, err := os.UserHomeDir()
		if err != nil {
			cobra.CheckErr(err)
		}
		settingFileDir = dirname
	}
	configHome := settingFileDir
	configName := ".backlog-git-pr-diff-checker"
	configType := "toml"

	viper.AddConfigPath(configHome)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	_ = viper.ReadInConfig()

	configPath := filepath.Join(configHome, configName+"."+configType)

	_, err := os.Stat(configPath)

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

var RootCmd = &cobra.Command{
	Version: defaults.Version,
	Use:     "backlog-git-pr-diff-checker",
	Short:   "It checks Git for particular path",
	Run:     runRootWrapper,
}

func Execute() error {
	RootCmd.PersistentFlags().StringSliceVarP(&targetPaths, "target-paths", "p", []string{}, "Target paths you want to filter.")
	RootCmd.PersistentFlags().StringVarP(&since, "since", "s", "", "Limit the commits to those made after the specified date.")
	RootCmd.PersistentFlags().StringVarP(&apiKey, "apikey", "k", "", "Backlog's api key")
	RootCmd.PersistentFlags().StringVarP(&description, "description", "d", "", "The name of this diff check")
	RootCmd.PersistentFlags().StringVarP(&repoDir, "repoDir", "r", "./", "The name of this diff check")
	RootCmd.PersistentFlags().StringVarP(&settingFileDir, "settingFileDir", "f", "", "The path of the setting file")

	RootCmd.Use = ""

	return RootCmd.Execute()
}

type OutputSection struct {
	Description  string
	Title        string
	RepoURL      string
	PullRequests []*backlog.PullRequest
}

var markdownTmplStr = `{{$ret := . -}}
{{.Description}}
# {{.Title}} 
{{range .PullRequests}}
* {{.Summary}} [#{{.Number}}]({{$ret.RepoURL}}/pullRequests/{{.Number}}) {{.CreatedUser.Name}} 
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
