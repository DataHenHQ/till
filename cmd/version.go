package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var (
	ReleaseVersion = "dev"
	ReleaseCommit  = "none"
	ReleaseDate    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of DataHen Till",
	Long:  `All software has versions. This is Till's`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("DataHen Till")
		fmt.Println("Version:", ReleaseVersion)
		fmt.Println("Date:", ReleaseDate)
		fmt.Println("Commit:", ReleaseCommit)
	},
}
