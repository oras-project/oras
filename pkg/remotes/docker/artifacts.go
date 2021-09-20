package docker

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	remoteserrors "github.com/containerd/containerd/remotes/errors"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func (d *dockerDiscoverer) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	d.reference = ref
	return d, nil
}

func (d *dockerDiscoverer) Push(ctx context.Context, desc ocispec.Descriptor) (content.Writer, error) {
	switch desc.MediaType {
	case artifactspec.MediaTypeArtifactManifest:
		h, err := d.filterHosts(docker.HostCapabilityPush)
		if err != nil {
			return nil, err
		}

		if len(h) == 0 {
			return nil, errors.New("no host with push")
		}

		host := h[0]

		err = d.CheckManifest(ctx, host, desc)
		if err != nil {
			return nil, err
		}

		return d.PreparePutManifest(ctx, host, desc)
	}

	pusher, err := d.Resolver.Pusher(ctx, d.reference)
	if err != nil {
		return nil, err
	}

	return pusher.Push(ctx, desc)
}

func (d *dockerDiscoverer) PreparePutManifest(ctx context.Context, host docker.RegistryHost, desc ocispec.Descriptor) (content.Writer, error) {
	d.tracker.SetStatus(d.reference, docker.Status{
		Status: content.Status{
			Ref:       d.reference,
			Total:     desc.Size,
			Expected:  desc.Digest,
			StartedAt: time.Now(),
		},
	})

	req := d.request(host, http.MethodPut, "manifests", desc.Digest.String())
	if req == nil {
		return nil, errors.New("not implemented")
	}

	req.header.Set("Content-Type", desc.MediaType)

	pr, pw := io.Pipe()
	respC := make(chan response, 1)
	body := ioutil.NopCloser(pr)

	req.body = func() (io.ReadCloser, error) {
		if body == nil {
			return nil, errors.New("retry request, cannot reset the stream")
		}

		readout := body
		body = nil
		return readout, nil
	}
	req.size = desc.Size

	go func() {
		defer close(respC)
		resp, err := req.doWithRetries(ctx, nil)
		if err != nil {
			respC <- response{err: err}
			pr.CloseWithError(err)
			return
		}

		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		default:
			err := remoteserrors.NewUnexpectedStatusErr(resp)
			pr.CloseWithError(err)
		}
		respC <- response{Response: resp}
	}()

	return &artifactsManifest{
		ref:       d.reference,
		expected:  desc.Digest,
		pipe:      pw,
		responseC: respC,
		tracker:   d.tracker,
	}, nil
}

func (d *dockerDiscoverer) CheckManifest(ctx context.Context, host docker.RegistryHost, desc ocispec.Descriptor) error {
	// Check if the manifest exists
	req := d.request(host, http.MethodHead, "manifests", desc.Digest.String())
	if req == nil {
		return errors.New("not implemented")
	}

	req.header.Set("Accept", desc.MediaType)

	resp, err := req.doWithRetries(ctx, nil)
	if err != nil {
		if !errors.Is(err, docker.ErrInvalidAuthorization) {
			return err
		}
	} else {
		if resp.StatusCode == http.StatusOK && resp.Header.Get("Docker-Content-Digest") == desc.Digest.String() {
			d.tracker.SetStatus(d.reference, docker.Status{
				Committed: true,
				Status: content.Status{
					Ref:    d.reference,
					Total:  desc.Size,
					Offset: desc.Size,
				}})

			return errdefs.ErrAlreadyExists
		} else if resp.StatusCode != http.StatusNotFound {
			err := remoteserrors.NewUnexpectedStatusErr(resp)
			resp.Body.Close()
			return err
		}
	}

	return nil
}

type response struct {
	*http.Response
	err error
}

type artifactsManifest struct {
	ref       string
	expected  digest.Digest
	pipe      *io.PipeWriter
	tracker   docker.StatusTracker
	responseC <-chan response
}

func (pw *artifactsManifest) Write(p []byte) (n int, err error) {
	status, err := pw.tracker.GetStatus(pw.ref)
	if err != nil {
		return n, err
	}
	n, err = pw.pipe.Write(p)
	status.Offset += int64(n)
	status.UpdatedAt = time.Now()
	pw.tracker.SetStatus(pw.ref, status)
	return
}

func (pw *artifactsManifest) Close() error {
	return pw.pipe.Close()
}

func (pw *artifactsManifest) Status() (content.Status, error) {
	status, err := pw.tracker.GetStatus(pw.ref)
	if err != nil {
		return content.Status{}, err
	}
	return status.Status, nil

}

func (pw *artifactsManifest) Digest() digest.Digest {
	return pw.expected
}

func (pw *artifactsManifest) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	// Check whether read has already thrown an error
	if _, err := pw.pipe.Write([]byte{}); err != nil && err != io.ErrClosedPipe {
		return errors.Wrap(err, "pipe error before commit")
	}

	if err := pw.pipe.Close(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-pw.responseC:
		if resp.err != nil {
			return resp.err
		}
		defer resp.Response.Body.Close()

		// 201 is specified return status, some registries return
		// 200, 202 or 204.
		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusNoContent, http.StatusAccepted:
		default:
			return remoteserrors.NewUnexpectedStatusErr(resp.Response)
		}

		status, err := pw.tracker.GetStatus(pw.ref)
		if err != nil {
			return errors.Wrap(err, "failed to get status")
		}

		if size > 0 && size != status.Offset {
			return errors.Errorf("unexpected size %d, expected %d", status.Offset, size)
		}

		if expected == "" {
			expected = status.Expected
		}

		actual, err := digest.Parse(resp.Header.Get("Docker-Content-Digest"))
		if err != nil {
			return errors.Wrap(err, "invalid content digest in response")
		}

		if actual != expected {
			return errors.Errorf("got digest %s, expected %s", actual, expected)
		}

		status.Committed = true
		status.UpdatedAt = time.Now()
		pw.tracker.SetStatus(pw.ref, status)
	}

	return nil
}

func (pw *artifactsManifest) Truncate(size int64) error {

	return errors.New("cannot truncate remote upload")
}

func requestWithMountFrom(req *request, mount, from string) *request {
	creq := *req

	sep := "?"
	if strings.Contains(creq.path, sep) {
		sep = "&"
	}

	creq.path = creq.path + sep + "mount=" + mount + "&from=" + from

	return &creq
}
