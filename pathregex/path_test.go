package pathregex

import (
	"reflect"
	"testing"
)

func TestCompilePath(t *testing.T) {
	type args struct {
		path          string
		caseSensitive bool
		end           bool
	}
	tests := []struct {
		args                 args
		param                []string
		expectedRegexPattern string
	}{
		{
			args: args{
				path:          "////aaaaa/:var1/:var2/*pathname",
				caseSensitive: true,
				end:           true,
			},
			param:                []string{"var1", "var2"},
			expectedRegexPattern: `^\/aaaaa\/([^\/]+)\/([^\/]+)\/\*pathname\/*$`,
		},
		{
			args: args{
				path:          "/cmd/:tool/:sub",
				caseSensitive: true,
				end:           true,
			},
			param:                []string{"tool", "sub"},
			expectedRegexPattern: `^\/cmd\/([^\/]+)\/([^\/]+)\/*$`,
		},
		{
			args: args{
				path:          "/order/:id",
				caseSensitive: true,
				end:           true,
			},
			param:                []string{"id"},
			expectedRegexPattern: `^\/order\/([^\/]+)\/*$`,
		},
		{
			args: args{
				path:          "/src/*filepath", // => only accept /src/* => * as variable
				caseSensitive: true,
				end:           true,
			},
			param:                []string{},
			expectedRegexPattern: `^\/src\/\*filepath\/*$`,
		},
		{
			args: args{
				path:          "/info/:user/project/:project",
				caseSensitive: true,
				end:           true,
			},
			param:                []string{"user", "project"},
			expectedRegexPattern: `^\/info\/([^\/]+)\/project\/([^\/]+)\/*$`,
		},
		{
			args: args{
				path:          "/info/hehe:user/project/:project",
				caseSensitive: true,
				end:           true,
			},
			param:                []string{"user", "project"},
			expectedRegexPattern: `^\/info\/hehe([^\/]+)\/project\/([^\/]+)\/*$`,
		},
		{
			args: args{
				path:          "/info/:user/project/:project/*",
				caseSensitive: true,
				end:           true,
			},
			param:                []string{"user", "project", "*"},
			expectedRegexPattern: `^\/info\/([^\/]+)\/project\/([^\/]+)(?:\/(.+)|\/*)$`,
		},
	}
	for _, tt := range tests {
		t.Run("testing match path with pattern...", func(t *testing.T) {
			res, param := CompilePath(tt.args.path, tt.args.caseSensitive, tt.args.end)
			if res.String() != tt.expectedRegexPattern {
				t.Errorf("CompilePath(%v,%v, %v) is = %v, expected %v", tt.args.path, tt.args.caseSensitive, tt.args.end, res, tt.expectedRegexPattern)
			}
			if !reflect.DeepEqual(param, tt.param) {
				t.Errorf("CompilePath(%v,%v, %v) is = %v, expected %v", tt.args.path, tt.args.caseSensitive, tt.args.end, param, tt.param)
			}
		})
	}
}

var emptyParam = make(map[string]string)

//   "/",
// "/cmd/:tool/:sub",
// "/cmd/:tool/",
// "/src/*filepath",
// "/search/",
// "/search/:query",
// "/user_:name",
// "/user_:name/about",
// "/files/:dir/*filepath",
// "/doc/",
// "/doc/go_faq.html",
// "/doc/go1.html",
// "/info/:user/public",
// "/info/:user/project/:project",
// {"/", false, "/", nil},
// 		{"/cmd/test/", false, "/cmd/:tool/", Params{Param{"tool", "test"}}},
// 		{"/cmd/test", true, "", Params{Param{"tool", "test"}}},
// 		{"/cmd/test/3", false, "/cmd/:tool/:sub", Params{Param{"tool", "test"}, Param{"sub", "3"}}},
// 		{"/src/", false, "/src/*filepath", Params{Param{"filepath", "/"}}},
// 		{"/src/some/file.png", false, "/src/*filepath", Params{Param{"filepath", "/some/file.png"}}},
// 		{"/search/", false, "/search/", nil},
// 		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
// 		{"/search/someth!ng+in+ünìcodé/", true, "", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
// 		{"/user_gopher", false, "/user_:name", Params{Param{"name", "gopher"}}},
// 		{"/user_gopher/about", false, "/user_:name/about", Params{Param{"name", "gopher"}}},
// 		{"/files/js/inc/framework.js", false, "/files/:dir/*filepath", Params{Param{"dir", "js"}, Param{"filepath", "/inc/framework.js"}}},
// 		{"/info/gordon/public", false, "/info/:user/public", Params{Param{"user", "gordon"}}},
// 		{"/info/gordon/project/go", false, "/info/:user/project/:project", Params{Param{"user", "gordon"}, Param{"project", "go"}}},

func TestMatchPath(t *testing.T) {
	type args struct {
		path    string
		pattern string
	}
	tests := []struct {
		args        args
		shouldMatch bool
		param       map[string]string
	}{
		{
			args: args{
				path:    "/cmd/test",
				pattern: "/cmd/:tool",
			},
			shouldMatch: true,
			param: map[string]string{
				"tool": "test",
			},
		},
		{
			args: args{
				path:    "/cmd/test/3",
				pattern: "/cmd/:tool/:sub",
			},
			shouldMatch: true,
			param: map[string]string{
				"tool": "test",
				"sub":  "3",
			},
		},
		{
			args: args{
				path:    "/info/gordon/public",
				pattern: "/info/:name/public",
			},
			shouldMatch: true,
			param: map[string]string{
				"name": "gordon",
			},
		},
		{
			args: args{
				path:    "/info/gordon/project/go",
				pattern: "/info/:name/project/:lang",
			},
			shouldMatch: true,
			param: map[string]string{
				"name": "gordon",
				"lang": "go",
			},
		},
		{
			args: args{
				path:    "/src/aaaaaaaaaaaaaaaa",
				pattern: "/src/*",
			},
			shouldMatch: true,
			param: map[string]string{
				"*": "aaaaaaaaaaaaaaaa",
			},
		},
	}
	for _, tt := range tests {
		t.Run("testing match path with pattern...", func(t *testing.T) {
			isMatch, param := MatchPath(tt.args.path, tt.args.pattern)
			if isMatch != tt.shouldMatch {
				t.Errorf("MatchPath(%v, %v) is match = %v, expected %v", tt.args.path, tt.args.pattern, isMatch, tt.shouldMatch)
			}
			if !reflect.DeepEqual(param, tt.param) {
				t.Errorf("MatchPath(%v, %v) param = %v, expected param %v", tt.args.path, tt.args.pattern, param, tt.param)
			}
		})
	}
}
