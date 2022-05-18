package option

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

type Credential struct {
	Configs           []string
	Username          string
	PasswordFromStdin bool
	Password          string
}

func (cred *Credential) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&cred.Configs, "config", "c", nil, "auth config path")
	fs.StringVarP(&cred.Username, "username", "u", "", "registry username")
	fs.StringVarP(&cred.Password, "password", "p", "", "registry password or identity token")
	fs.BoolVarP(&cred.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
}

func (cred *Credential) Prompt() (err error) {
	if cred.Password != "" {
		fmt.Fprintln(os.Stderr, "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	} else if cred.PasswordFromStdin {
		// Prompt for credential
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		cred.Password = strings.TrimSuffix(string(password), "\n")
		cred.Password = strings.TrimSuffix(cred.Password, "\r")
	}
	return nil
}
