package cmd

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/1984weed/backlog-git-pr-diff-checker/defaults"

	"github.com/1984weed/backlog-git-pr-diff-checker/exit"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	since       string
	apiKey      string
	targetPaths []string
)

func RunRoot(cmd *cobra.Command, args []string) (string, error) {
	initViper()

	myDirHash := getMyDirHash()

	var s map[string]struct {
		LastCommit string `toml:"lastcommit"`
	}

	_ = viper.Unmarshal(&s)

	prevCommit := ""
	if val, ok := s[myDirHash]; ok {
		prevCommit = val.LastCommit
	}

	if since != "" {
		out, err := exec.Command("bash", "-c", fmt.Sprintf("git log --since='%s' %s", since, "--pretty=format:%H | tail -n 1")).Output()
		if err != nil {
			return "", err
		}
		prevCommit = string(out)
	}

	if prevCommit == "" {
		return "", errors.New("It must be set since flag for first.")
	}

	targetPathsStr := []string{}
	for _, v := range targetPaths {
		targetPathsStr = append(targetPathsStr, fmt.Sprintf("--target-paths=%s", v))
	}

	gbchArgs := []string{fmt.Sprintf("--apikey=%s", apiKey),
		fmt.Sprintf("--from=%s", prevCommit),
		"-F=markdown"}
	gbchArgs = append(gbchArgs, targetPathsStr...)

	out, err := exec.Command("gbch", gbchArgs...).Output()

	if err != nil {
		return "", err
	}

	lastCommitOut, _ := exec.Command("bash", "-c", "git log --pretty=format:%H | head -n 1").Output()
	viper.Set(fmt.Sprintf("%s.lastCommit", myDirHash), strings.TrimSuffix(string(lastCommitOut), "\n"))

	err = viper.WriteConfig()

	if err != nil {
		return "", err
	}

	return string(out), nil
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

	rootCmd.Use = ""

	return rootCmd.Execute()
}
