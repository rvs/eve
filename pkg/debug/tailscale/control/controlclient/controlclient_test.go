// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package controlclient

import (
	"reflect"
	"testing"

	"tailscale.com/types/empty"
)

func fieldsOf(t reflect.Type) (fields []string) {
	for i := 0; i < t.NumField(); i++ {
		fields = append(fields, t.Field(i).Name)
	}
	return
}

func TestStatusEqual(t *testing.T) {
	// Verify that the Equal method stays in sync with reality
	equalHandles := []string{"LoginFinished", "Err", "URL", "Persist", "NetMap", "Hostinfo", "state"}
	if have := fieldsOf(reflect.TypeOf(Status{})); !reflect.DeepEqual(have, equalHandles) {
		t.Errorf("Status.Equal check might be out of sync\nfields: %q\nhandled: %q\n",
			have, equalHandles)
	}

	tests := []struct {
		a, b *Status
		want bool
	}{
		{
			&Status{},
			nil,
			false,
		},
		{
			nil,
			&Status{},
			false,
		},
		{
			&Status{},
			&Status{},
			true,
		},
		{
			&Status{state: stateNew},
			&Status{state: stateNew},
			true,
		},
		{
			&Status{state: stateNew},
			&Status{state: stateAuthenticated},
			false,
		},
		{
			&Status{LoginFinished: nil},
			&Status{LoginFinished: new(empty.Message)},
			false,
		},
	}
	for i, tt := range tests {
		got := tt.a.Equal(tt.b)
		if got != tt.want {
			t.Errorf("%d. Equal = %v; want %v", i, got, tt.want)
		}
	}
}
