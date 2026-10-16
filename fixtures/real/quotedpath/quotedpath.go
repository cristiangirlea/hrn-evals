// Package quotedpath reads the program out of a path in a command line, as
// hrn's run viewer does to shorten C:\Program Files\Git\bin\bash.exe to
// bash.exe. The command line may be plain text or JSON text, where a
// backslash is written \\ and a double quote \".
package quotedpath

import "strings"

// Program reads the path that starts at s[i] and continues past a system
// folder at s[j], up to a quote, a space or a delimiter, and returns where it
// ends and the program's file name; "" when it does not name a program. In
// quotes a Windows folder may have spaces in its name (Microsoft VS Code) and
// punctuation (Git (Preview)), so only a quote or a line end stops it; a
// Windows program has an extension, which tells it from a folder.
//
// i is where the path starts (just after its opening quote, when it has one)
// and j where the part after the system folder starts: for
// "C:\Program Files\Git\bin\bash.exe" -lc ls, i is 1 and j is the index of
// the G of Git.
func Program(s string, i, j int, windows bool) (int, string) {
	var quote byte
	stops := "\"'`\r\n,;|&<>()[]{}"
	if i > 0 && (s[i-1] == '"' || s[i-1] == '\'') {
		quote, stops = s[i-1], "\"'`\r\n"
	}
	end, last := j, j // last: where the current element starts
	for ; end < len(s); end++ {
		c := s[end]
		if c == '/' || windows && c == '\\' {
			last = end + 1
			continue
		}
		if c == ' ' || c == '\t' {
			if quote != 0 && windows && !strings.Contains(s[last:end], ".") {
				continue
			}
			break
		}
		if strings.IndexByte(stops, c) >= 0 {
			break
		}
	}
	for end > j && s[end-1] == '\\' { // a JSON-escaped closing quote's backslash
		end--
	}
	name := s[j:end]
	if k := strings.LastIndexAny(name, `/\`); k >= 0 && (windows || name[k] == '/') {
		name = name[k+1:]
	}
	if name == "" || windows && !strings.Contains(name, ".") || strings.ContainsAny(name, `/\`) {
		return 0, ""
	}
	return end, name
}
