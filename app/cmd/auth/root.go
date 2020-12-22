package auth

import (
	"io"
	"os"

	"github.com/Frizz925/gilgamesh/auth"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authorization management",
	}
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newDeleteCmd())
	return cmd
}

func readCredentials(filename string) (auth.Credentials, error) {
	if filename == "" || filename == "-" {
		return make(auth.Credentials), nil
	}
	f, err := os.Open(filename)
	if os.IsNotExist(err) {
		return make(auth.Credentials), nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()
	return auth.ReadCredentials(f)
}

func writeCredentials(filename string, credentials auth.Credentials) error {
	var w io.Writer = os.Stdout
	if filename != "" && filename != "-" {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return auth.WriteCredentials(w, credentials)
}
