package main

import (
	"github.com/wraix/device-flow-proxy/cmd"
)

var (
	name string = "device-flow-proxy"
	version string = "0.0.0"
	commit string
	date string
	tag string
)

func main() {
	cmd.Execute(name, version, commit, date, tag)
}
