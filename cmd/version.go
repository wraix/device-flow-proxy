package cmd

import (
	"fmt"

	"github.com/wraix/device-flow-proxy/app"
)

type versionCmd struct {
	// version   bool `short:"v" long:"version" description:"display version"`
}

func (v *versionCmd) Execute(args []string) error {
  fmt.Printf("Name: %s\nVersion: %s\nTag: %s\nCommit: %s\nDate: %s\n", app.Env.Build.Name, app.Env.Build.Version, app.Env.Build.Tag, app.Env.Build.Commit, app.Env.Build.Date)
	return nil
}
