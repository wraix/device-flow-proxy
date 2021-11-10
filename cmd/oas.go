package cmd

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/charmixer/oas/exporter"
	"github.com/wraix/device-flow-proxy/router"
)

type oasCmd struct {
	// version   bool `short:"v" long:"version" description:"display version"`
}

func (v *oasCmd) Execute(args []string) error {
	router := router.NewRouter(Application.Name, Application.Description, Application.Version)

	oasModel := exporter.ToOasModel(router.OpenAPI)
	oasYaml, err := yaml.Marshal(&oasModel)
	if err != nil {
		return err
	}

	fmt.Println(string(oasYaml))

	return nil
}
