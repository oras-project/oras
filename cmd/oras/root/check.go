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

package root

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/oras-project/oras-go/v3/errdef"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
)

// defaultCheckConcurrency is the number of blobs checked at the same time
// when --concurrency is not set to a positive value. It matches the default
// used by oras-go's VerifyGraph.
const defaultCheckConcurrency = 8

type checkOptions struct {
	option.Common
	option.Platform
	option.Target

	concurrency int
}

func checkCmd() *cobra.Command {
	var opts checkOptions

	cmd := &cobra.Command{
		Use:   "check [flags] [<name>[:<tag>|@<digest>]]",
		Short: "[Experimental] Verify that an artifact or OCI image layout is complete and intact",
		Long: `[Experimental] Verify that an artifact or OCI image layout is complete and intact

The check reads the manifest, config and every layer of the artifact, in a
registry or an OCI image layout, and proves that each one is present and
matches its digest and size. Every byte is downloaded or read. When an OCI
image layout is given without a tag or digest, the whole layout is checked, as
a directory or a tar archive. The check continues after a failure and reports
every problem it finds, and the command exits with a non-zero status if any
check fails.

The check only proves that content matches its digests. It does not detect that
a tag was moved to other content, and it does not prove who published the
content. A blob that a registry cut short may be reported as unable to check
rather than as corrupt.

Example - Check an artifact in a registry:
  oras check localhost:5000/hello-world:v1

Example - Check an artifact by digest:
  oras check localhost:5000/hello-world@sha256:0a4d6bd4e0a1a3a5a0f1e5c0b0c4e3d2a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4

Example - Check an artifact in an OCI image layout:
  oras check --oci-layout ./layout-dir:v1

Example - Check a whole OCI image layout directory or tar archive:
  oras check --oci-layout ./layout-dir
  oras check --oci-layout ./layout.tar
  oras check --oci-layout-path ./layout-dir

Example - Check the image for one platform of a multi-platform artifact:
  oras check --platform linux/arm64 localhost:5000/hello-world:v1
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.Path != "" {
				return oerrors.CheckArgs(argument.AtMost(1), "the artifact reference to check")(cmd, args)
			}
			return oerrors.CheckArgs(argument.Exactly(1), "the artifact reference to check")(cmd, args)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RawReference = args[0]
			}
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.Printer.Verbose = true // most nodes have no title, and PrintStatus hides them otherwise
			return runCheck(cmd, &opts)
		},
	}

	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", defaultCheckConcurrency, "number of blobs checked at the same time; 0 or less uses the default")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runCheck(cmd *cobra.Command, opts *checkOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	isLayout := opts.Type == option.TargetTypeOCILayout
	if isLayout && opts.Reference == "" && (!opts.IsOCILayout || opts.Path == opts.RawReference) {
		if opts.Platform.Platform != nil {
			return errors.New("--platform cannot be used when checking a whole OCI image layout")
		}
		return doCheckWholeLayout(ctx, opts.Path, opts.concurrency, opts.Printer)
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}

	var src oras.ReadOnlyTarget
	if isLayout {
		layout, err := newCheckLayout(opts.Path)
		if err != nil {
			return fmt.Errorf("invalid argument %q: %w", opts.RawReference, err)
		}
		src = layout
	} else {
		repo, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
		if err != nil {
			return err
		}
		src = repo
	}

	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = opts.Platform.Platform
	return doCheck(ctx, src, opts.Reference, resolveOpts, isLayout, opts.concurrency, opts.Printer)
}

// checkClass is the kind of problem found in a node.
type checkClass int

// The classes are ordered by how the summary lists them.
const (
	classMissing checkClass = iota
	classCorrupt
	classInvalid
	classUnchecked
)

func (c checkClass) String() string {
	switch c {
	case classMissing:
		return "Missing"
	case classCorrupt:
		return "Corrupt"
	case classInvalid:
		return "Invalid"
	default:
		return "Unable to check"
	}
}

// errCheckInvalid marks content that is present but not a valid artifact node.
var errCheckInvalid = errors.New("invalid content")

// checkFinding is one failed node or layout problem and the reason.
type checkFinding struct {
	desc  ocispec.Descriptor
	path  string
	class checkClass
	err   error
}

func (f checkFinding) label() string {
	switch {
	case f.desc.Digest != "" && f.desc.MediaType != "":
		return f.desc.MediaType + " " + f.desc.Digest.String()
	case f.desc.Digest != "":
		return f.desc.Digest.String()
	default:
		return f.path
	}
}

// checker holds the state shared by the concurrent callbacks of one check.
type checker struct {
	root     ocispec.Descriptor
	isLayout bool
	printer  *output.Printer

	mu sync.Mutex
	// bySubject and byOther record how each digest was reached: through a
	// subject edge, or through any other edge. They are written when the
	// parent is read, which always happens before the child is checked.
	bySubject map[digest.Digest]bool
	byOther   map[digest.Digest]bool
	// foreign holds the digests of foreign layers, which oras-go drops from
	// the walk.
	foreign map[digest.Digest]bool
}

func doCheck(ctx context.Context, src oras.ReadOnlyTarget, reference string, resolveOpts oras.ResolveOptions, isLayout bool, concurrency int, printer *output.Printer) error {
	root, err := oras.Resolve(ctx, src, reference, resolveOpts)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", reference, err)
	}

	c := &checker{
		root:      root,
		isLayout:  isLayout,
		printer:   printer,
		bySubject: make(map[digest.Digest]bool),
		byOther:   map[digest.Digest]bool{root.Digest: true},
		foreign:   make(map[digest.Digest]bool),
	}
	if concurrency <= 0 {
		concurrency = defaultCheckConcurrency
	}
	report, err := oras.VerifyGraph(ctx, src, root, oras.VerifyGraphOptions{
		Concurrency:    concurrency,
		FindSuccessors: c.findSuccessors,
		PostVerify: func(_ context.Context, desc ocispec.Descriptor, verifyErr error) error {
			status := "Verified"
			if verifyErr != nil {
				status = "Failed"
			}
			return printer.PrintStatus(desc, status)
		},
	})
	// A walk that stopped early returns an error and no summary, because the
	// report is then incomplete and must not be read as a pass.
	if err != nil {
		return fmt.Errorf("check stopped before it finished: %w", err)
	}
	return c.summarize(report)
}

func doCheckWholeLayout(ctx context.Context, layoutPath string, concurrency int, printer *output.Printer) error {
	info, err := os.Stat(layoutPath)
	if err != nil {
		return fmt.Errorf("invalid argument %q: %w", layoutPath, err)
	}
	if concurrency <= 0 {
		concurrency = defaultCheckConcurrency
	}
	if info.IsDir() {
		return doCheckLayoutDir(ctx, layoutPath, concurrency, printer)
	}
	return doCheckLayoutTar(ctx, layoutPath, concurrency, printer)
}

// doCheckLayoutDir verifies a whole OCI image layout directory by combining
// oci.Verify (which sweeps blobs/, layout files and reachability) with the
// manifest graph walk on every entry in index.json.
func doCheckLayoutDir(ctx context.Context, layoutPath string, concurrency int, printer *output.Printer) error {
	fsys := os.DirFS(layoutPath)
	ociReport, err := oci.Verify(ctx, fsys, oci.VerifyOptions{Concurrency: concurrency})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("check stopped before it finished: %w", err)
		}
		return fmt.Errorf("%q is not an OCI image layout: %w", layoutPath, err)
	}

	// Seed VerifiedSet from blobs oci.Verify already hashed, using their sizes
	// on disk so the manifest walk skips re-reading intact layers while still
	// catching descriptor size mismatches.
	verifiedSet := oras.NewVerifiedSet()
	verifiedDigests := make(map[digest.Digest]bool, len(ociReport.Verified))
	for _, entry := range ociReport.Verified {
		dgst, ok := blobPathDigest(entry.Path)
		if !ok {
			continue
		}
		verifiedDigests[dgst] = true
		if info, err := fs.Stat(fsys, entry.Path); err == nil {
			verifiedSet.Add(ocispec.Descriptor{Digest: dgst, Size: info.Size()})
		}
	}

	c := &checker{
		isLayout:  true,
		printer:   printer,
		bySubject: make(map[digest.Digest]bool),
		byOther:   make(map[digest.Digest]bool),
		foreign:   make(map[digest.Digest]bool),
	}

	var (
		walkResult             layoutWalkResult
		walkInvalidIndexDigest bool
	)
	if raw, err := os.ReadFile(filepath.Join(layoutPath, ocispec.ImageIndexFile)); err == nil {
		var index ocispec.Index
		if err := json.Unmarshal(raw, &index); err == nil {
			for _, desc := range index.Manifests {
				if desc.Digest.Validate() != nil {
					walkInvalidIndexDigest = true
				}
			}
			storage := oci.NewStorageFromFS(fsys)
			walkResult, err = c.walkLayoutManifests(ctx, storage, index.Manifests, verifiedSet, concurrency)
			if err != nil {
				return err
			}
		}
	}

	failures := walkResult.failures
	warnings := walkResult.warnings
	failedDigests := walkResult.failedDigests
	if failedDigests == nil {
		failedDigests = make(map[digest.Digest]bool)
	}
	for dgst := range walkResult.verified {
		verifiedDigests[dgst] = true
	}

	for _, entry := range ociReport.Failed {
		if dgst, ok := blobPathDigest(entry.Path); ok {
			if failedDigests[dgst] {
				continue
			}
			failedDigests[dgst] = true
			failures = append(failures, checkFinding{
				desc:  ocispec.Descriptor{Digest: dgst},
				class: c.classify(entry.Reason),
				err:   entry.Reason,
			})
			if !walkResult.printedFailed[dgst] {
				if err := printer.PrintStatus(ocispec.Descriptor{Digest: dgst}, "Failed"); err != nil {
					return err
				}
			}
		} else {
			failures = append(failures, checkFinding{
				path:  entry.Path,
				class: c.classify(entry.Reason),
				err:   entry.Reason,
			})
		}
	}

	seenUnverifiable := make(map[string]bool)
	for _, entry := range ociReport.Unverifiable {
		if dgst, ok := blobPathDigest(entry.Path); ok {
			if failedDigests[dgst] {
				continue
			}
			failedDigests[dgst] = true
			failures = append(failures, checkFinding{
				desc:  ocispec.Descriptor{Digest: dgst},
				class: classUnchecked,
				err:   entry.Reason,
			})
			continue
		}
		if entry.Path == ocispec.ImageIndexFile && errors.Is(entry.Reason, errdef.ErrInvalidDigest) && walkInvalidIndexDigest {
			continue
		}
		key := entry.Path + "\x00" + entry.Reason.Error()
		if seenUnverifiable[key] {
			continue
		}
		seenUnverifiable[key] = true
		failures = append(failures, checkFinding{
			path:  entry.Path,
			class: classUnchecked,
			err:   entry.Reason,
		})
	}

	var leftovers []oci.LayoutFinding
	for _, lf := range ociReport.Layout {
		if lf.ModTime.IsZero() {
			failures = append(failures, checkFinding{
				path:  lf.Path,
				class: classInvalid,
				err:   lf.Reason,
			})
		} else {
			leftovers = append(leftovers, lf)
		}
	}
	sort.Slice(leftovers, func(i, j int) bool {
		return leftovers[i].Path < leftovers[j].Path
	})

	// A blob that matched its path digest on disk may still have failed the
	// manifest walk (for example, wrong descriptor size or invalid manifest
	// shape), so remove any failed digest from the verified count.
	for dgst := range failedDigests {
		delete(verifiedDigests, dgst)
	}
	unprintedVerified := make([]digest.Digest, 0)
	for dgst := range verifiedDigests {
		if !walkResult.printedVerified[dgst] && !walkResult.printedFailed[dgst] {
			unprintedVerified = append(unprintedVerified, dgst)
		}
	}
	sort.Slice(unprintedVerified, func(i, j int) bool {
		return unprintedVerified[i] < unprintedVerified[j]
	})
	for _, dgst := range unprintedVerified {
		if err := printer.PrintStatus(ocispec.Descriptor{Digest: dgst}, "Verified"); err != nil {
			return err
		}
	}

	unreferenced := 0
	for _, entry := range ociReport.Unreferenced {
		if dgst, ok := blobPathDigest(entry.Path); ok && failedDigests[dgst] {
			continue
		}
		unreferenced++
	}

	sortFindings(failures)
	sortFindings(warnings)
	return c.printSummary(len(verifiedDigests), failures, warnings, leftovers, unreferenced, false)
}

// doCheckLayoutTar verifies every entry in index.json of an OCI image layout
// tar archive without extracting it.
func doCheckLayoutTar(ctx context.Context, layoutPath string, concurrency int, printer *output.Printer) error {
	layout, err := newCheckLayout(layoutPath)
	if err != nil {
		return fmt.Errorf("%q is not an OCI image layout: %w", layoutPath, err)
	}

	c := &checker{
		isLayout:  true,
		printer:   printer,
		bySubject: make(map[digest.Digest]bool),
		byOther:   make(map[digest.Digest]bool),
		foreign:   make(map[digest.Digest]bool),
	}
	walkResult, err := c.walkLayoutManifests(ctx, layout, layout.manifests, oras.NewVerifiedSet(), concurrency)
	if err != nil {
		return err
	}

	sortFindings(walkResult.failures)
	sortFindings(walkResult.warnings)
	return c.printSummary(len(walkResult.verified), walkResult.failures, walkResult.warnings, nil, 0, true)
}

type layoutWalkResult struct {
	verified        map[digest.Digest]bool
	failures        []checkFinding
	warnings        []checkFinding
	failedDigests   map[digest.Digest]bool
	printedVerified map[digest.Digest]bool
	printedFailed   map[digest.Digest]bool
}

// walkLayoutManifests walks every descriptor in manifests, sharing verifiedSet
// across calls so shared leaf blobs are not read more than once.
func (c *checker) walkLayoutManifests(ctx context.Context, src content.ReadOnlyStorage, manifests []ocispec.Descriptor, verifiedSet *oras.VerifiedSet, concurrency int) (layoutWalkResult, error) {
	result := layoutWalkResult{
		verified:        make(map[digest.Digest]bool),
		failedDigests:   make(map[digest.Digest]bool),
		printedVerified: make(map[digest.Digest]bool),
		printedFailed:   make(map[digest.Digest]bool),
	}
	c.mu.Lock()
	for _, desc := range manifests {
		if desc.Digest != "" {
			// Every entry listed directly in index.json is required even if
			// another manifest also points to it as a subject.
			c.byOther[desc.Digest] = true
		}
	}
	c.mu.Unlock()

	var (
		printMu   sync.Mutex
		rawFailed []oras.FailedNode
	)
	printOnce := func(desc ocispec.Descriptor, failed bool) error {
		printMu.Lock()
		defer printMu.Unlock()
		if failed {
			if desc.Digest != "" && result.printedFailed[desc.Digest] {
				return nil
			}
			if desc.Digest != "" {
				result.printedFailed[desc.Digest] = true
			}
			return c.printer.PrintStatus(desc, "Failed")
		}
		if desc.Digest != "" && (result.printedVerified[desc.Digest] || result.printedFailed[desc.Digest]) {
			return nil
		}
		if desc.Digest != "" {
			result.printedVerified[desc.Digest] = true
		}
		return c.printer.PrintStatus(desc, "Verified")
	}

	for _, root := range manifests {
		report, err := oras.VerifyGraph(ctx, src, root, oras.VerifyGraphOptions{
			Concurrency:    concurrency,
			KnownVerified:  verifiedSet,
			FindSuccessors: c.findSuccessors,
			PostVerify: func(_ context.Context, desc ocispec.Descriptor, verifyErr error) error {
				return printOnce(desc, verifyErr != nil)
			},
			OnVerifySkipped: func(_ context.Context, desc ocispec.Descriptor) error {
				return printOnce(desc, false)
			},
		})
		if err != nil {
			return layoutWalkResult{}, fmt.Errorf("check stopped before it finished: %w", err)
		}
		for _, desc := range report.Verified {
			if desc.Digest != "" {
				result.verified[desc.Digest] = true
			}
		}
		for _, desc := range report.Skipped {
			if desc.Digest != "" {
				result.verified[desc.Digest] = true
			}
		}
		rawFailed = append(rawFailed, report.Failed...)
	}

	seenWarnings := make(map[digest.Digest]bool)
	for _, failed := range rawFailed {
		finding := checkFinding{desc: failed.Descriptor, class: c.classify(failed.Err), err: failed.Err}
		if c.isAbsentSubject(finding) {
			if !seenWarnings[finding.desc.Digest] {
				seenWarnings[finding.desc.Digest] = true
				result.warnings = append(result.warnings, finding)
			}
			continue
		}
		if finding.desc.Digest != "" {
			if result.failedDigests[finding.desc.Digest] {
				continue
			}
			result.failedDigests[finding.desc.Digest] = true
		}
		result.failures = append(result.failures, finding)
	}
	for dgst := range result.failedDigests {
		delete(result.verified, dgst)
	}
	return result, nil
}

// blobPathDigest extracts the digest from a layout blob path of the form
// blobs/<algorithm>/<encoded>.
func blobPathDigest(p string) (digest.Digest, bool) {
	rest, ok := strings.CutPrefix(p, ocispec.ImageBlobsDir+"/")
	if !ok {
		return "", false
	}
	alg, encoded, ok := strings.Cut(rest, "/")
	if !ok || strings.Contains(encoded, "/") {
		return "", false
	}
	dgst := digest.NewDigestFromEncoded(digest.Algorithm(alg), encoded)
	if err := dgst.Validate(); err != nil {
		return "", false
	}
	return dgst, true
}

// findSuccessors reads a manifest or index, checks that its body matches the
// media type of the descriptor that points to it, and returns every node it
// references, including its config and its subject. Other nodes have no
// successors and are checked by reading them in full.
func (c *checker) findSuccessors(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	if !isCheckManifest(desc.MediaType) {
		return nil, nil
	}
	if desc.Size > maxCheckManifestSize {
		return nil, fmt.Errorf("a manifest of %d bytes is larger than the limit of %d bytes, so it was not read", desc.Size, maxCheckManifestSize)
	}
	body, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return nil, err
	}
	if err := checkManifestShape(desc.MediaType, body); err != nil {
		return nil, err
	}
	nodes, subject, config, err := graph.Successors(ctx, fetcher, desc)
	if err != nil {
		return nil, err
	}

	successors := make([]ocispec.Descriptor, 0, len(nodes)+2)
	successors = append(successors, nodes...)
	if config != nil {
		successors = append(successors, *config)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, node := range nodes {
		c.byOther[node.Digest] = true
		if isForeignLayer(node.MediaType) {
			c.foreign[node.Digest] = true
		}
	}
	if config != nil {
		c.byOther[config.Digest] = true
	}
	if subject != nil {
		successors = append(successors, *subject)
		c.bySubject[subject.Digest] = true
	}
	return successors, nil
}

// summarize prints the result in a stable order and returns an error if any
// node failed.
func (c *checker) summarize(report *oras.VerifyGraphReport) error {
	var failures []checkFinding
	var warnings []checkFinding
	for _, failed := range report.Failed {
		finding := checkFinding{desc: failed.Descriptor, class: c.classify(failed.Err), err: failed.Err}
		if c.isAbsentSubject(finding) {
			warnings = append(warnings, finding)
			continue
		}
		failures = append(failures, finding)
	}
	sortFindings(failures)
	sortFindings(warnings)
	return c.printSummary(len(report.Verified), failures, warnings, nil, 0, false)
}

func (c *checker) printSummary(verified int, failures, warnings []checkFinding, leftovers []oci.LayoutFinding, unreferenced int, isTar bool) error {
	// An absent subject is not a node of the target, so it is not counted.
	total := verified + len(failures)
	if err := c.printer.Println(fmt.Sprintf("Checked %d nodes: %d verified, %d failed", total, verified, len(failures))); err != nil {
		return err
	}
	for _, finding := range failures {
		if err := c.printer.Println(fmt.Sprintf("%s: %s: %s", finding.class, finding.label(), finding.err)); err != nil {
			return err
		}
	}
	for _, finding := range warnings {
		if err := c.printer.Println(fmt.Sprintf("Warning: the subject %s is not in the target, which the OCI image spec allows", finding.desc.Digest)); err != nil {
			return err
		}
	}
	c.mu.Lock()
	foreign := len(c.foreign)
	c.mu.Unlock()
	if foreign > 0 {
		if err := c.printer.Println(fmt.Sprintf("Not checked: %d foreign layers, which are not stored in the target", foreign)); err != nil {
			return err
		}
	}
	if isTar {
		if err := c.printer.Println("Not checked: unreferenced blobs and stray files are not checked in a tar archive"); err != nil {
			return err
		}
	}
	for _, lf := range leftovers {
		if err := c.printer.Println(fmt.Sprintf("Note: %s is a leftover from an interrupted write (modified %s)", lf.Path, lf.ModTime.UTC().Format(time.RFC3339))); err != nil {
			return err
		}
	}
	if unreferenced == 1 {
		if err := c.printer.Println("Note: 1 unreferenced blob is not reachable from any tag (reliable only when nothing is writing to the layout)"); err != nil {
			return err
		}
	} else if unreferenced > 1 {
		if err := c.printer.Println(fmt.Sprintf("Note: %d unreferenced blobs are not reachable from any tag (reliable only when nothing is writing to the layout)", unreferenced)); err != nil {
			return err
		}
	}
	if len(failures) > 0 {
		// A plain error: it must not wrap errdef.ErrNotFound or an
		// errcode.ErrorResponse, which Target.ModifyError would present as
		// a registry error.
		return fmt.Errorf("check failed: %d of %d nodes did not pass", len(failures), total)
	}
	return nil
}

// isAbsentSubject reports whether a failed node is a subject that is simply
// not in the target. The OCI image spec allows that, so it is only a warning.
// A subject that is present but fails, a node that is also reached another way,
// and the artifact that was asked for are always failures.
func (c *checker) isAbsentSubject(f checkFinding) bool {
	if f.class != classMissing || f.desc.Digest == c.root.Digest {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bySubject[f.desc.Digest] && !c.byOther[f.desc.Digest]
}

// classify names the problem behind an error. Only content that is proven
// wrong is called corrupt; anything uncertain is unable to check.
func (c *checker) classify(err error) checkClass {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.Is(err, errdef.ErrNotFound):
		return classMissing
	case errors.Is(err, content.ErrMismatchedDigest), errors.Is(err, content.ErrTrailingData):
		return classCorrupt
	// A short read from a registry can be a dropped connection, so it is
	// proof of corruption only for a local layout.
	case errors.Is(err, io.ErrUnexpectedEOF) && c.isLayout:
		return classCorrupt
	case errors.Is(err, errCheckInvalid), errors.Is(err, errdef.ErrInvalidDigest),
		errors.As(err, &syntaxErr), errors.As(err, &typeErr),
		strings.Contains(err.Error(), "invalid successor descriptor"):
		return classInvalid
	default:
		return classUnchecked
	}
}

func sortFindings(findings []checkFinding) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.class != b.class {
			return a.class < b.class
		}
		if a.desc.MediaType != b.desc.MediaType {
			return a.desc.MediaType < b.desc.MediaType
		}
		if a.desc.Digest != b.desc.Digest {
			return a.desc.Digest < b.desc.Digest
		}
		return a.path < b.path
	})
}

func isCheckManifest(mediaType string) bool {
	return isCheckIndex(mediaType) || isCheckImageManifest(mediaType)
}

func isCheckIndex(mediaType string) bool {
	return mediaType == ocispec.MediaTypeImageIndex || mediaType == docker.MediaTypeManifestList
}

func isCheckImageManifest(mediaType string) bool {
	return mediaType == ocispec.MediaTypeImageManifest || mediaType == docker.MediaTypeManifest ||
		mediaType == graph.MediaTypeArtifactManifest
}

// isForeignLayer reports whether a layer is a foreign layer. These are the
// exact media types that oras-go drops from the walk; any other type is
// verified, not skipped.
func isForeignLayer(mediaType string) bool {
	switch mediaType {
	case "application/vnd.oci.image.layer.nondistributable.v1.tar",
		"application/vnd.oci.image.layer.nondistributable.v1.tar+gzip",
		"application/vnd.oci.image.layer.nondistributable.v1.tar+zstd",
		"application/vnd.docker.image.rootfs.foreign.diff.tar.gzip":
		return true
	}
	return false
}

// checkManifestShape compares the body of a manifest with the media type of the
// descriptor that points to it. The mediaType field of a manifest is optional,
// so when it is absent the shape of the body decides: an index has manifests,
// an image manifest has a config or blobs.
func checkManifestShape(mediaType string, body []byte) error {
	var fields struct {
		MediaType string          `json:"mediaType"`
		Manifests json.RawMessage `json:"manifests"`
		Config    json.RawMessage `json:"config"`
		Layers    json.RawMessage `json:"layers"`
		Blobs     json.RawMessage `json:"blobs"`
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	if fields.MediaType != "" && fields.MediaType != mediaType {
		return fmt.Errorf("%w: the body says its media type is %s, but it is described as %s", errCheckInvalid, fields.MediaType, mediaType)
	}
	hasIndexShape := fields.Manifests != nil
	hasImageShape := fields.Config != nil || fields.Layers != nil || fields.Blobs != nil
	switch {
	case isCheckIndex(mediaType) && (!hasIndexShape || hasImageShape):
		return fmt.Errorf("%w: it is described as %s but its body is not an index", errCheckInvalid, mediaType)
	case isCheckImageManifest(mediaType) && (hasIndexShape || (!hasImageShape && fields.MediaType == "")):
		return fmt.Errorf("%w: it is described as %s but its body is not a manifest", errCheckInvalid, mediaType)
	}
	return nil
}

// checkLayout reads an OCI image layout, in a directory or in a tar archive,
// without loading its graph. That way one corrupt manifest is reported as a
// failed node, and does not stop an intact tag next to it from being checked.
type checkLayout struct {
	content.ReadOnlyStorage
	manifests []ocispec.Descriptor
}

func newCheckLayout(layoutPath string) (*checkLayout, error) {
	info, err := os.Stat(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find path %q: %w", layoutPath, err)
	}
	var (
		storage content.ReadOnlyStorage
		raw     []byte
	)
	if info.IsDir() {
		storage = oci.NewStorageFromFS(os.DirFS(layoutPath))
		raw, err = os.ReadFile(filepath.Join(layoutPath, ocispec.ImageIndexFile))
	} else {
		storage, err = oci.NewStorageFromTar(layoutPath)
		if err == nil {
			raw, err = readTarFile(layoutPath, ocispec.ImageIndexFile)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%q is not an OCI image layout: %w", layoutPath, err)
	}
	var index ocispec.Index
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("%q has an %s that cannot be read: %w", layoutPath, ocispec.ImageIndexFile, err)
	}
	return &checkLayout{ReadOnlyStorage: storage, manifests: index.Manifests}, nil
}

// maxCheckManifestSize is the largest manifest or index that check reads into
// memory, and the most of a blob that is read to find out what a digest
// reference points to. It matches the default limit in oras-go.
const maxCheckManifestSize = 4 * 1024 * 1024

// Resolve finds a manifest in the top-level index by its tag, or by its digest.
// A digest that is not listed in the top-level index, such as a manifest nested
// in an index, is looked up in the blob storage.
func (l *checkLayout) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	// The last match wins, as it does when oras-go loads the index.
	for i := len(l.manifests) - 1; i >= 0; i-- {
		desc := l.manifests[i]
		if desc.Digest.String() == reference || desc.Annotations[ocispec.AnnotationRefName] == reference {
			return desc, nil
		}
	}
	if dgst, err := digest.Parse(reference); err == nil {
		return l.resolveBlob(ctx, dgst)
	}
	return ocispec.Descriptor{}, fmt.Errorf("%s: %w", reference, errdef.ErrNotFound)
}

// resolveBlob builds a descriptor for a blob that the top-level index does not
// list. The media type comes from the body of the manifest; a blob that is not
// a manifest is described as a plain blob, and is checked against its digest.
func (l *checkLayout) resolveBlob(ctx context.Context, dgst digest.Digest) (ocispec.Descriptor, error) {
	desc := ocispec.Descriptor{MediaType: "application/octet-stream", Digest: dgst}
	rc, err := l.Fetch(ctx, desc)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer rc.Close()
	body, err := io.ReadAll(io.LimitReader(rc, maxCheckManifestSize+1))
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	desc.Size = int64(len(body))
	if desc.Size > maxCheckManifestSize {
		// Too big to read as a manifest. If it looks like JSON it may still be
		// one, and checking only its bytes would skip everything it references.
		if strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
			return ocispec.Descriptor{}, fmt.Errorf("%s is larger than the limit of %d bytes and may be a manifest", dgst, maxCheckManifestSize)
		}
		// Not a manifest; count the rest to get the real size.
		rest, err := io.Copy(io.Discard, rc)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		desc.Size += rest
		return desc, nil
	}
	var fields struct {
		MediaType string          `json:"mediaType"`
		Manifests json.RawMessage `json:"manifests"`
		Config    json.RawMessage `json:"config"`
		Layers    json.RawMessage `json:"layers"`
		Blobs     json.RawMessage `json:"blobs"`
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return desc, nil
	}
	switch {
	case isCheckManifest(fields.MediaType):
		desc.MediaType = fields.MediaType
	case fields.Manifests != nil:
		desc.MediaType = ocispec.MediaTypeImageIndex
	case fields.Config != nil && fields.Layers != nil:
		desc.MediaType = ocispec.MediaTypeImageManifest
	}
	return desc, nil
}

// readTarFile reads one file from a tar archive. The tar file system of oras-go
// is internal, so the index has to be read here.
func readTarFile(archive, name string) ([]byte, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := tar.NewReader(f)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: %w", name, errdef.ErrNotFound)
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg && path.Clean(header.Name) == name {
			return io.ReadAll(reader)
		}
	}
}
