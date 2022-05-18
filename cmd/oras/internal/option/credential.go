/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package option

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// Credential options struct.
type Credential struct {
	Configs           []string
	Username          string
	PasswordFromStdin bool
	Password          string
}

// ApplyFlags applies flags to a command flag set.
func (cred *Credential) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&cred.Configs, "config", "c", nil, "auth config path")
	fs.StringVarP(&cred.Username, "username", "u", "", "registry username")
	fs.StringVarP(&cred.Password, "password", "p", "", "registry password or identity token")
	fs.BoolVarP(&cred.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
}

// ReadPassword tries to read password with optional cmd prompt.
func (cred *Credential) ReadPassword() (err error) {
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
