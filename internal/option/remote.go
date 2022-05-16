package option

import "github.com/spf13/pflag"

type Remote struct {
	Hostname   string
	Configs    []string
	CaFilePath string
	FromStdin  bool
	Username   string
	Password   string
	PlainHTTP  bool
	Insecure   bool
}

func (opts *Remote) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&opts.Configs, "config", "c", nil, "auth config path")
	fs.StringVarP(&opts.Username, "username", "u", "", "registry username")
	fs.StringVarP(&opts.Password, "password", "p", "", "registry password or identity token")
	fs.BoolVarP(&opts.FromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	fs.BoolVarP(&opts.Insecure, "insecure", "k", false, "allow connections to SSL registry without certs")
	fs.StringVarP(&opts.CaFilePath, "ca-file", "", "", "server certificate authority file for the remote registry")
	fs.BoolVarP(&opts.PlainHTTP, "plain-http", "", false, "allow insecure connections to registry without SSL")
}
