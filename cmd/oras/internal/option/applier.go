package option

import (
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type applier interface {
	ApplyFlagsTo(*pflag.FlagSet)
}

func ApplyFlags(opts interface{}, target *cobra.Command) {
	v := reflect.ValueOf(opts)
	for i := 0; i < v.NumField(); i++ {
		iface := v.Field(i).Interface()
		if a, ok := iface.(applier); ok {
			a.ApplyFlagsTo(target.Flags())
		}
	}
}
