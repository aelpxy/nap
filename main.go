package main

import "github.com/aelpxy/yap/cmd"

var (
	Version   = "alpha"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, BuildTime, GitCommit)
	cmd.Execute()
}
