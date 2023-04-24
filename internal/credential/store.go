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

package credential

import credentials "github.com/oras-project/oras-credentials-go"

func NewStore(configPaths ...string) (credentials.Store, error) {
	var stores []credentials.Store
	so := credentials.StoreOptions{AllowPlaintextPut: true}
	for _, config := range configPaths {
		store, err := credentials.NewStore(config, so)
		if err != nil {
			return nil, err
		}
		stores = append(stores, store)
	}
	return credentials.NewStoreWithFallbacks(stores[0], stores[1:]...), nil
}
