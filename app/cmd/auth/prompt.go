package auth

import (
	"errors"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
)

var (
	usernameInput = &survey.Input{Message: "Username:"}
	passwordInput = &survey.Password{Message: "Password:"}
)

func usernameValidator(v interface{}) error {
	if err := survey.Required(v); err != nil {
		return err
	}
	ans, ok := v.(string)
	if !ok {
		return errors.New("prompt answer is not a string")
	}
	match, err := regexp.MatchString("^[a-zA-Z0-9]+$", ans)
	if err != nil {
		return err
	}
	if !match {
		return errors.New("username must be alphanumeric")
	}
	return nil
}

func getUsernamePassword(args []string) (username string, password string, err error) {
	nargs := len(args)
	if nargs < 1 {
		err = survey.AskOne(usernameInput, &username, survey.WithValidator(usernameValidator))
		if err != nil {
			return
		}
	} else {
		username = args[0]
	}
	if nargs < 2 {
		err = survey.AskOne(passwordInput, &password)
		if err != nil {
			return
		}
	} else {
		password = args[1]
	}
	return username, password, nil
}
