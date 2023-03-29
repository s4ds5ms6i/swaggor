package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCMD = &cobra.Command{
	Use:   "swaggor",
	Short: "An OpenAPI (Swagger) generator for Golang",
	Long:  `Swaggor is an OpenAPI (Swagger) generator for Golang, while (echo) acts as a web service.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCMD.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCMD.AddCommand(generateCMD)
}
