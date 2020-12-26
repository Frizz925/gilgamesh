package auth

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Credentials map[string]Password

type Password []byte

func CreatePassword(plaintext []byte) (Password, error) {
	b, err := bcrypt.GenerateFromPassword(plaintext, bcrypt.DefaultCost)
	return Password(b), err
}

func (p Password) Compare(password []byte) error {
	return bcrypt.CompareHashAndPassword(p[:], password)
}

func WriteCredentials(w io.Writer, credentials Credentials) error {
	var bw *bufio.Writer
	if v, ok := w.(*bufio.Writer); ok {
		bw = v
	} else {
		bw = bufio.NewWriter(w)
	}
	for user, password := range credentials {
		line := fmt.Sprintf("%s:%s\n", user, password)
		if _, err := bw.WriteString(line); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func ReadCredentials(r io.Reader) (Credentials, error) {
	sc := bufio.NewScanner(r)
	result := make(Credentials)
	for sc.Scan() {
		parts := strings.Split(sc.Text(), ":")
		user, password := parts[0], Password(parts[1])
		result[user] = password
	}
	return result, nil
}
