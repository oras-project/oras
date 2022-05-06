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

package credential_test

import (
	"context"
	"math/rand"
	"path/filepath"
	"strconv"
	"testing"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/credential"
)

func TestStore_storeGetErase(t *testing.T) {
	tempDir := t.TempDir()
	confFileName := "test.json"
	configPath := filepath.Join(tempDir, confFileName)
	regName := "test"
	cred := auth.Credential{
		Username: "username",
		Password: "password",
	}

	// store
	store, err := credential.NewStore(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err.Error())
	}
	err = store.Store(regName, cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// get cred
	got, err := store.Credential(context.Background(), regName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != cred {
		t.Fatalf("expect: %v, got: %v", cred, got)
	}

	// erase
	err = store.Erase(regName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err = store.Credential(context.Background(), regName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != auth.EmptyCredential {
		t.Fatalf("expect: %v, got: %v", auth.EmptyCredential, got)
	}
}

func TestStore_getEmptyCred(t *testing.T) {
	store, err := credential.NewStore()
	if err != nil {
		t.Fatalf("Failed to create store with default config path: %v", err.Error())
	}

	got, err := store.Credential(context.Background(), strconv.Itoa(rand.Int()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != auth.EmptyCredential {
		t.Fatalf("expect: %v, got: %v", auth.EmptyCredential, got)
	}
}
