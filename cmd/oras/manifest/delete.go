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
	"fmt"

	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/errors"
)

type deleteOptions struct {
	option.Common
	option.Remote

	targetRef string
}

func deleteManifest(opts deleteOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}

	ref := opts.targetRef
	desc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return err
	}
	if err = repo.Delete(ctx, desc); err != nil {
		return err
	}

	fmt.Println("Deleted", opts.targetRef)

	return nil
}
