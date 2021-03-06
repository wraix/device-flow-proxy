package cmd

import (
	"fmt"

	"gopkg.in/yaml.v2"

  "github.com/wraix/device-flow-proxy/app"
	"github.com/charmixer/oas/exporter"
	"github.com/wraix/device-flow-proxy/router"
)

type oasCmd struct {}

func (v *oasCmd) Execute(args []string) error {
	router := router.NewRouter(app.Env.Build.Name, Application.Description, app.Env.Build.Version)

	oasModel := exporter.ToOasModel(router.OpenAPI)
	oasYaml, err := yaml.Marshal(&oasModel)
	if err != nil {
		return err
	}

	fmt.Println(string(oasYaml))

	return nil
}
