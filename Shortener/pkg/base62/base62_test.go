package base62

import (
	"reflect"
	"testing"
)

func TestInt2String(t *testing.T) {
	type args struct {
		seq uint64
	}
	tests := []struct {
		name string
		seq  uint64
		want string
	}{
		{name: "1", seq: 1, want: "1"},
		{name: "2", seq: 61, want: "Z"},
		{name: "3", seq: 62, want: "10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int2String(tt.seq); got != tt.want {
				t.Errorf("Int2String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString2Int(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		s    string
		want uint64
	}{
		{name: "1", s: "10", want: 62},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String2Int(tt.s); got != tt.want {
				t.Errorf("String2Int() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reverse(t *testing.T) {
	type args struct {
		s []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reverse(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reverse() = %v, want %v", got, tt.want)
			}
		})
	}
}
