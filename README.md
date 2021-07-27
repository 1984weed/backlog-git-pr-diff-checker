# backlog-git-pr-diff-checker

# Installation

```
go get  github.com/trknhr/backlog-git-pr-diff-checker
```

# Usage

It check specific paths defined by --target-paths. You move to your target repository's path.
Then you run the command like this.

```
backlog-git-pr-diff-checker --apikey=xxxx --since="5 days ago" --target-paths="./css,./js" --description="The new pull requests are under the /src/css/"
```

# Options

```
  -k, --apikey string          Backlog's api key
  -h, --help                   help for this command
  -s, --since string           Limit the commits to those made after the specified date.
  -p, --target-paths strings   Target paths you want to filter.
```
