package option

import (
	"github.com/spf13/pflag"
)

type Applier interface {
	ApplyFlagsTo(*pflag.FlagSet)
}
