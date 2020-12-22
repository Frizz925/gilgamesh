package auth

import (
	"github.com/Frizz925/gilgamesh/auth"

	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [filename] [username] [password]",
		Short: "Create or update user in the passwords file",
		Args:  cobra.MaximumNArgs(3),
		RunE:  runSetCmd,
	}
}

func runSetCmd(cmd *cobra.Command, args []string) error {
	var filename string
	if len(args) > 0 {
		filename = args[0]
		args = args[1:]
	}
	username, password, err := getUsernamePassword(args)
	if err != nil {
		return err
	}
	pw, err := auth.CreatePassword([]byte(password))
	if err != nil {
		return err
	}
	creds, err := readCredentials(filename)
	if err != nil {
		return err
	}
	creds[username] = pw
	return writeCredentials(filename, creds)
}
