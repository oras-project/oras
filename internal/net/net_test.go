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
	"net"
	"reflect"
	"testing"
)

func TestDialer_DialContext(t *testing.T) {
	type args struct {
		ctx     context.Context
		network string
		addr    string
	}
	tests := []struct {
		name    string
		d       *Dialer
		args    args
		want    net.Conn
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.DialContext(tt.args.ctx, tt.args.network, tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Dialer.DialContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Dialer.DialContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemote_parseResolve_ipv4(t *testing.T) {
	host := "mockedHost"
	port := "12345"
	address := "192.168.1.1"
	var d Dialer
	d.Add(host, 12345, net.ParseIP(address))

	if len(d.resolve) != 1 {
		t.Fatalf("expect 1 resolve entries but got %v", d.resolve)
	}
	want := make(map[string]string)
	want[host+":"+port] = address + ":" + port
	if !reflect.DeepEqual(want, d.resolve) {
		t.Fatalf("expecting %v  but got %v", want, d.resolve)
	}
}
