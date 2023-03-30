package cmd

import (
	stdlog "log"
	"os"

	"github.com/spf13/cobra"

	"gitlab.snapp.ir/security_regulatory/swaggor/config"
	"gitlab.snapp.ir/security_regulatory/swaggor/log"
)

var (
	cfgFile string
	rootCMD = &cobra.Command{
		Use:   "swaggor",
		Short: "An OpenAPI (Swagger) generator for Golang",
		Long:  `Swaggor is an OpenAPI (Swagger) generator for Golang, while (echo) acts as a web service.`,
	}
)

func Execute() {
	err := rootCMD.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	err := config.ParseConfig(cfgFile)
	if err != nil {
		stdlog.Fatalf("config file parse failed: %s", err)
	}

	log.Init(config.C.Log.Level, config.C.Log.Path)

	rootCMD.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default is ./config.yaml)")

	rootCMD.AddCommand(generateCMD)
}
