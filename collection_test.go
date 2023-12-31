package mockhttp

import (
	"reflect"
	"testing"
)

func Test_in(t *testing.T) {
	type args struct {
		current     string
		collections []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not exist",
			args: args{
				current:     "test",
				collections: []string{"test-1", "test-2", "test-3"},
			},
			want: false,
		},
		{
			name: "exist in collection",
			args: args{
				current:     "test-1",
				collections: []string{"test-1", "test-2", "test-3"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := in(tt.args.current, tt.args.collections); got != tt.want {
				t.Errorf("in() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_some(t *testing.T) {
	type args struct {
		collections []int
		fn          func(int) bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"one of the element satisfy the condition",
			args{
				collections: []int{1, 2, 3},
				fn:          func(v int) bool { return v == 3 },
			},
			true,
		},
		{
			"no element satisfy the condition",
			args{
				collections: []int{1, 2, 3},
				fn:          func(v int) bool { return v > 3 },
			},
			false,
		},
		{
			"all elements satisfy the condition",
			args{
				collections: []int{1, 2, 3},
				fn:          func(v int) bool { return v <= 3 },
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := some(tt.args.collections, tt.args.fn); got != tt.want {
				t.Errorf("some() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_all(t *testing.T) {
	type args struct {
		collection []int
		fn         func(int) bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"one of the element satisfy the condition",
			args{
				collection: []int{1, 2, 3},
				fn:         func(v int) bool { return v == 3 },
			},
			false,
		},
		{
			"no element satisfy the condition",
			args{
				collection: []int{1, 2, 3},
				fn:         func(v int) bool { return v > 3 },
			},
			false,
		},
		{
			"all elements satisfy the condition",
			args{
				collection: []int{1, 2, 3},
				fn:         func(v int) bool { return v <= 3 },
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := all(tt.args.collection, tt.args.fn); got != tt.want {
				t.Errorf("all() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findFirst(t *testing.T) {
	type args struct {
		collections []string
		fn          func(string) bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"find first element with length 1",
			args{
				collections: []string{"a", "b", "c"},
				fn:          func(val string) bool { return len(val) == 1 },
			},
			"a",
			false,
		},
		{
			"not find any element that satisfy collection",
			args{
				collections: []string{"a", "b", "c"},
				fn:          func(val string) bool { return len(val) > 1 },
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findFirst(tt.args.collections, tt.args.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("findFirst() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findFirst() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_merge(t *testing.T) {
	type args struct {
		collections [][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"merge all string",
			args{
				collections: [][]string{{"1", "2", "3"}, {"1", "2", "3"}},
			},
			[]string{"1", "2", "3", "1", "2", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := merge(tt.args.collections...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
