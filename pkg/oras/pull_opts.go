package oras

type pullOpts struct {
	allowedMediaTypes []string
}

// PullOpt allows callers to set options on the oras pull
type PullOpt func(o *pullOpts) error

func pullOptsDefaults() *pullOpts {
	return &pullOpts{}
}

// WithAllowedMediaTypes sets the allowed media types
func WithAllowedMediaTypes(allowedMediaTypes ...string) PullOpt {
	return func(o *pullOpts) error {
		o.allowedMediaTypes = append(o.allowedMediaTypes, allowedMediaTypes...)
		return nil
	}
}
