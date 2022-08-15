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

package manifest

import (
	"errors"
	"fmt"

	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/file"
	"oras.land/oras/cmd/oras/internal/option"
)

type pushOptions struct {
	option.Common
	option.Remote

	targetRef string
	fileRef   string
	mediaType string
}

func pushManifest(opts pushOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	var mediaType string
	if opts.mediaType != "" {
		mediaType = opts.mediaType
	} else {
		mediaType, err = file.ParseMediaType(opts.fileRef)
		if err != nil {
			return errors.New("media type cannot be recognized")
		}
	}

	// prepare manifest content
	desc, rc, err := file.PrepareContent(opts.fileRef, mediaType)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := rc.Close(); err == nil {
			err = closeErr
		}
	}()

	exists, err := repo.Exists(ctx, desc)
	if err != nil {
		return err
	}
	if exists {
		statusPrinter := display.StatusPrinter("Exists   ", opts.Verbose)
		if err := statusPrinter(ctx, desc); err != nil {
			return err
		}
	} else {
		if err = repo.Push(ctx, desc, rc); err != nil {
			return err
		}
	}

	if tag := repo.Reference.Reference; tag != "" {
		repo.Tag(ctx, desc, tag)
	}

	fmt.Println("Pushed", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
