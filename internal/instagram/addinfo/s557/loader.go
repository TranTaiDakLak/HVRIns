package addinfo

import (
	"bufio"
	"math/rand"
	"os"
	"strings"
)

// entry holds a name loaded from a config file.
// For schools.txt: Name is the school name to search for.
// For cities.txt / hometowns.txt: files are optional fallback (not required).
type entry struct {
	Name string
}

func loadEntries(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []entry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Support both "name" and legacy "pageID|name" formats; take last token.
		parts := strings.SplitN(line, "|", 2)
		name := strings.TrimSpace(parts[len(parts)-1])
		if name != "" {
			out = append(out, entry{Name: name})
		}
	}
	return out, sc.Err()
}

func pickRandom(entries []entry) (entry, bool) {
	if len(entries) == 0 {
		return entry{}, false
	}
	return entries[rand.Intn(len(entries))], true
}
