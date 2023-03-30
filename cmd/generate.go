package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"gitlab.snapp.ir/security_regulatory/swaggor/handler"
)

var (
	rootDir         string
	excludedDirsStr string
	outputDir       string
	excludedDirs    []string
)

var generateCMD = &cobra.Command{
	Use:   "generate",
	Short: "Generate OpenAPI",
	Run: func(cmd *cobra.Command, args []string) {
		if len(excludedDirsStr) > 0 {
			excludedDirs = strings.Split(excludedDirsStr, ",")
		}

		handler.Generate(rootDir, outputDir, excludedDirs)
	},
}

func init() {
	generateCMD.Flags().StringVarP(&rootDir, "project-root", "r", "", "the root directory of the project")
	generateCMD.Flags().StringVarP(&excludedDirsStr, "exclude", "e", "", "the directories that must be excluded in comma separated form")
	generateCMD.Flags().StringVarP(&outputDir, "output", "o", "", "the output directory for OpenAPI")
}
