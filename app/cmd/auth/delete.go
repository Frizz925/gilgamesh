package auth

import "github.com/spf13/cobra"

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <filename> <username>",
		Short: "Delete user from passwords file",
		Args:  cobra.ExactArgs(2),
		RunE:  runDeleteCmd,
	}
}

func runDeleteCmd(cmd *cobra.Command, args []string) error {
	filename, username := args[0], args[1]
	creds, err := readCredentials(filename)
	if err != nil {
		return err
	}
	delete(creds, username)
	return writeCredentials(filename, creds)
}
