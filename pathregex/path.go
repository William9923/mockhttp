package pathregex

import (
	"regexp"
	"strings"
)

// CleanPath is the URL version of path.Clean, it returns a canonical URL path
// for p, eliminating . and .. elements.
//
// The following rules are applied iteratively until no further processing can
// be done:
//  1. Replace multiple slashes with a single slash.
//  2. Eliminate each . path name element (the current directory).
//  3. Eliminate each inner .. path name element (the parent directory)
//     along with the non-.. element that precedes it.
//  4. Eliminate .. elements that begin a rooted path:
//     that is, replace "/.." by "/" at the beginning of a path.
//
// If the result of this process is an empty string, "/" is returned
func CleanPath(p string) string {
	const stackBufSize = 128

	// Turn empty string into "/"
	if p == "" {
		return "/"
	}

	// Reasonably sized buffer on stack to avoid allocations in the common case.
	// If a larger buffer is required, it gets allocated dynamically.
	buf := make([]byte, 0, stackBufSize)

	n := len(p)

	// Invariants:
	//      reading from path; r is index of next byte to process.
	//      writing to buf; w is index of next byte to write.

	// path must start with '/'
	r := 1
	w := 1

	if p[0] != '/' {
		r = 0

		if n+1 > stackBufSize {
			buf = make([]byte, n+1)
		} else {
			buf = buf[:n+1]
		}
		buf[0] = '/'
	}

	trailing := n > 1 && p[n-1] == '/'

	// A bit more clunky without a 'lazybuf' like the path package, but the loop
	// gets completely inlined (bufApp calls).
	// So in contrast to the path package this loop has no expensive function
	// calls (except make, if needed).

	for r < n {
		switch {
		case p[r] == '/':
			// empty path element, trailing slash is added after the end
			r++

		case p[r] == '.' && r+1 == n:
			trailing = true
			r++

		case p[r] == '.' && p[r+1] == '/':
			// . element
			r += 2

		case p[r] == '.' && p[r+1] == '.' && (r+2 == n || p[r+2] == '/'):
			// .. element: remove to last /
			r += 3

			if w > 1 {
				// can backtrack
				w--

				if len(buf) == 0 {
					for w > 1 && p[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}

		default:
			// Real path element.
			// Add slash if needed
			if w > 1 {
				bufApp(&buf, p, w, '/')
				w++
			}

			// Copy element
			for r < n && p[r] != '/' {
				bufApp(&buf, p, w, p[r])
				w++
				r++
			}
		}
	}

	// Re-append trailing slash
	if trailing && w > 1 {
		bufApp(&buf, p, w, '/')
		w++
	}

	// If the original string was not modified (or only shortened at the end),
	// return the respective substring of the original string.
	// Otherwise return a new string from the buffer.
	if len(buf) == 0 {
		return p[:w]
	}
	return string(buf[:w])
}

// Internal helper to lazily create a buffer if necessary.
// Calls to this function get inlined.
func bufApp(buf *[]byte, s string, w int, c byte) {
	b := *buf
	if len(b) == 0 {
		// No modification of the original string so far.
		// If the next character is the same as in the original string, we do
		// not yet have to allocate a buffer.
		if s[w] == c {
			return
		}

		// Otherwise use either the stack buffer, if it is large enough, or
		// allocate a new buffer on the heap, and copy all previous characters.
		if l := len(s); l > cap(b) {
			*buf = make([]byte, len(s))
		} else {
			*buf = (*buf)[:l]
		}
		b = *buf

		copy(b, s[:w])
	}
	b[w] = c
}

// CompilePath compile usual HTTP endpoint path to a canonical regex based path
// for categorizing exact endpoint path, wildcard and path params.
//
// The following process are applied:
//
//  1. Remove / ignore trailing / and /*
//
//  2. Adding a leading / to ensure canonical path had leading /
//
//  3. Remove all special meta character in the path (via regex QuoteMeta)
//
//  4. Escape all / into \\/
//
//  5. Extract all path param (ex: /path/:id => id is path param)
//
//  6. Also extract if wildcards exist in path (ex: /path/*)
//
//     It output the canonical regular expression to match the path, and the path param names
func CompilePath(path string, caseSensitive bool, end bool) (*regexp.Regexp, []string) {

	regexpSource := regexp.MustCompile(`\/*\*?$`).ReplaceAllString(path, "")
	regexpSource = regexp.MustCompile(`^\/*`).ReplaceAllString(regexpSource, "/")
	regexpSource = regexp.QuoteMeta(regexpSource)
	regexpSource = strings.ReplaceAll(regexpSource, "/", "\\/")

	paramsRe := regexp.MustCompile(`:(\w+)`)
	matches := paramsRe.FindAllStringSubmatch(regexpSource, -1)
	paramNames := make([]string, len(matches))
	for i, match := range matches {
		paramNames[i] = match[0][1:]
	}
	regexpSource = paramsRe.ReplaceAllString(regexpSource, "([^\\/]+)")

	regexpSource = "^" + regexpSource
	if strings.HasSuffix(path, "*") {
		paramNames = append(paramNames, "*")
		if path == "*" || path == "/*" {
			regexpSource += "(.*)$"
		} else {
			regexpSource += "(?:\\/(.+)|\\/*)$"
		}
	} else if end {
		regexpSource += "\\/*$"
	} else if path != "" && path != "/" {
		regexpSource += "(?:(?=\\/|$))"
	}
	prefix := ""
	if !caseSensitive {
		prefix = "(?i)"
	}
	matcher := regexp.MustCompile(prefix + regexpSource)
	return matcher, paramNames
}

// MatchPath applies mathing between HTTP path and canonical path
// to check whether the pattern match / not.
//
//	It output the matching result (boolean), and the path param resolved values
func MatchPath(path string, pattern string) (bool, map[string]string) {
	matcher, paramNames := CompilePath(CleanPath(pattern), true, true)

	res := matcher.FindStringSubmatch(path)
	if res == nil {
		return false, nil
	}

	if len(paramNames) == 0 {
		return true, make(map[string]string)
	}

	if len(paramNames) != len(res)-1 {
		return false, nil
	}

	params := make(map[string]string)

	for idx, parseRes := range res[1:] {
		params[paramNames[idx]] = parseRes
	}

	return true, params
}
