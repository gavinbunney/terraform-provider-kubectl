package kubernetes

import (
	"reflect"
	"testing"
)

func Test_expandStringSlice(t *testing.T) {
	type args struct {
		s []interface{}
	}
	tests := []struct {
		name  string
		given []interface{}
		then  []string
	}{
		{
			"validate non empty strings not mutated",
			[]interface{}{"one", "two"},
			[]string{"one", "two"},
		},
		{
			"validate nil elements are empty strings",
			[]interface{}{nil, "two"},
			[]string{"", "two"},
		},
		{
			"validate empty array",
			[]interface{}{},
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandStringSlice(tt.given); !reflect.DeepEqual(got, tt.then) {
				t.Errorf("expandStringSlice() = %v, want %v", got, tt.then)
			}
		})
	}
}
