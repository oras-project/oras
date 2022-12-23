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

package net

import (
	"context"
	"fmt"
	"net"
)

// Dialer struct provides dialing function with predefined DNS resolves.
type Dialer struct {
	*net.Dialer
	resolve map[string]string
}

// Add adds an entry for DNS resolve.
func (d *Dialer) Add(from string, port int, to net.IP) {
	if d.resolve == nil {
		d.resolve = make(map[string]string)
	}
	d.resolve[fmt.Sprintf("%s:%d", from, port)] = fmt.Sprintf("%s:%d", to, port)
}

// DialContext connects to the addr on the named network using
// the provided context.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if resolve, ok := d.resolve[addr]; ok {
		addr = resolve
	}
	return d.Dialer.DialContext(ctx, network, addr)
}
