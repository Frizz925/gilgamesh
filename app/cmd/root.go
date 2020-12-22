package cmd

import (
	"fmt"
	"os"

	"github.com/Frizz925/gilgamesh/app/cmd/auth"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gilgamesh",
		Short: "Gilgamesh is a high-performance web proxy",
		Run:   runServeCmd,
	}
	cmd.AddCommand(auth.NewCmd())
	return cmd
}

func Execute() {
	cmd := NewCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
}
