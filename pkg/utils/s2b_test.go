package utils

import (
	"reflect"
	"testing"
)

func TestString2Byte(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name  string
		args  args
		wantB []byte
	}{
		{
			name: "test-1",
			args: args{
				raw: "a",
			},
			wantB: []byte{'a'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotB := String2Byte(tt.args.raw); !reflect.DeepEqual(gotB, tt.wantB) {
				t.Errorf("String2byte() = %v, want %v", gotB, tt.wantB)
			}
		})
	}
}
