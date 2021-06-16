# backlog-git-pr-diff-checker

# Requires 

This tool uses [gbch](https://github.com/vvatanabe/gbch). You need install it.

```
go get github.com/vvatanabe/gbch/cmd/gbch/
```

# Installation

```
go get  github.com/trknhr/backlog-git-pr-diff-checker
```

# Usage

It check specific paths defined by --target-paths. You move to your target repository's path.
Then you run the command like this.

```
backlog-git-pr-diff-checker --apikey=xxxx --since="5 days ago" --target-paths="./css,./js"
```

# Options

```
  -k, --apikey string          Backlog's api key
  -h, --help                   help for this command
  -s, --since string           Limit the commits to those made after the specified date.
  -p, --target-paths strings   Target paths you want to filter.
```
