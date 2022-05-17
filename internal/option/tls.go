package option

import "github.com/spf13/pflag"

type TLS struct {
	CaFilePath string
	PlainHTTP  bool
	Insecure   bool
}

func (tls TLS) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVarP(&tls.Insecure, "insecure", "k", false, "allow connections to SSL registry without certs")
	fs.StringVarP(&tls.CaFilePath, "ca-file", "", "", "server certificate authority file for the remote registry")
	fs.BoolVarP(&tls.PlainHTTP, "plain-http", "", false, "allow insecure connections to registry without SSL")
}
