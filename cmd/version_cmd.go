package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string

var versionCmd = cobra.Command{
	Run: showVersion,
	Use: "version",
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Version)
}
