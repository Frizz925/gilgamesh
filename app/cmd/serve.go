package cmd

import (
	"github.com/Frizz925/gilgamesh/app/server"
	"github.com/spf13/cobra"
)

func runServeCmd(cmd *cobra.Command, args []string) {
	if err := server.Start(); err != nil {
		panic(err)
	}
}
