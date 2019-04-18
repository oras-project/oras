package oras

import "github.com/containerd/containerd/images"

type pullOpts struct {
	allowedMediaTypes []string
	inSequence        bool
	baseHandlers      []images.Handler
}

// PullOpt allows callers to set options on the oras pull
type PullOpt func(o *pullOpts) error

func pullOptsDefaults() *pullOpts {
	return &pullOpts{}
}

// WithAllowedMediaType sets the allowed media types
func WithAllowedMediaType(allowedMediaTypes ...string) PullOpt {
	return func(o *pullOpts) error {
		o.allowedMediaTypes = append(o.allowedMediaTypes, allowedMediaTypes...)
		return nil
	}
}

// WithAllowedMediaTypes sets the allowed media types
func WithAllowedMediaTypes(allowedMediaTypes []string) PullOpt {
	return func(o *pullOpts) error {
		o.allowedMediaTypes = append(o.allowedMediaTypes, allowedMediaTypes...)
		return nil
	}
}

// WithPullInSequence opt to pull in sequence with breath-first search
func WithPullInSequence(o *pullOpts) error {
	o.inSequence = true
	return nil
}

// WithPullBaseHandler provides base handlers, which will be called before
// any pull specific handlers.
func WithPullBaseHandler(handlers ...images.Handler) PullOpt {
	return func(o *pullOpts) error {
		o.baseHandlers = append(o.baseHandlers, handlers...)
		return nil
	}
}
