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
	"time"
)

var defaultDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

type Dialer struct {
	resolve map[string]string
}

func (d *Dialer) Add(from string, port int, to net.IP) {
	if d.resolve == nil {
		d.resolve = make(map[string]string)
	}
	d.resolve[fmt.Sprintf("%s:%d", from, port)] = fmt.Sprintf("%s:%d", to, port)
}

// DialContext connects to the addr on the named network using
// the provided context.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	for k := range d.resolve {
		if k == addr {
			addr = d.resolve[k]
		}
	}
	return defaultDialer.DialContext(ctx, network, addr)
}
