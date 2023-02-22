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
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestRemote_parseResolve_ipv4(t *testing.T) {
	host := "mockedHost"
	hostPort := 443
	address := "192.168.1.1"
	addressPort := 12345
	var d Dialer
	d.Add(host, hostPort, net.ParseIP(address), addressPort)

	if len(d.resolve) != 1 {
		t.Fatalf("expect 1 resolve entries but got %v", d.resolve)
	}
	want := make(map[string]string)
	want[host+":"+fmt.Sprint(hostPort)] = address + ":" + fmt.Sprint(addressPort)
	if !reflect.DeepEqual(want, d.resolve) {
		t.Fatalf("expecting %v  but got %v", want, d.resolve)
	}
}
