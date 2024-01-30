package util

import (
	"runtime"
	"strings"
)

func FunctionName(depth int) string {
	pc, _, _, _ := runtime.Caller(depth)
	return strings.TrimPrefix(runtime.FuncForPC(pc).Name(), "runtime.")
}

// ChunkText takes in an arbitrarily sized chunk of text, splits on whitespace, and returns in chunks of 2000 chars or less
// When possible, it will group by paragraphs
func ChunkText(text string) []string {
	var chunks []string
	for len(text) > 0 {
		if len(text) <= 2000 {
			chunks = append(chunks, text)
			break
		}
		end := 2000
		if i := strings.LastIndex(text[:2000], "\n"); i != -1 {
			end = i
		} else if i := strings.LastIndex(text[:2000], " "); i != -1 {
			end = i
		}
		chunks = append(chunks, text[:end])
		text = text[end:]
		if text[0] == '\n' || text[0] == ' ' {
			text = text[1:] // Remove the leading newline/space from the remaining text
		}
	}
	return chunks
}
