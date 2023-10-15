package ignore

import (
	_ "embed"
	"flag"
	"os"
	"strings"
	"sync"

	pm "github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

//go:embed default.txt
var DefaultIgnores string
var ignoreArg string
var patternMatcher *pm.PatternMatcher 
var mu sync.Mutex

func init() {
	flag.StringVar(&ignoreArg, "ignore", "", "ignore file to use")
}

func InitIgnores() {
    r := strings.NewReader(DefaultIgnores)
    lines, err := ignorefile.ReadAll(r)
    if err != nil {
        panic(err)
    }

    if ignoreArg != "" {
        r, err := os.Open(ignoreArg)
        if err != nil {
            panic(err)
        }
        defer r.Close()

        moreLines, err := ignorefile.ReadAll(r)
        if err != nil {
            panic(err)
        }

        lines = append(lines, moreLines...)
    }

    patternMatcher, err = pm.New(lines)
    if err != nil {
        panic(err)
    }
}

func Match(path string) bool {
    mu.Lock()
    defer mu.Unlock()

    matches, err := patternMatcher.MatchesOrParentMatches(path)
    if err != nil {
        panic(err)
    }
    return matches
}
