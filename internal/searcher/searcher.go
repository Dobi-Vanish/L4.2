package searcher

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
)

type Searcher struct {
	pattern     *regexp.Regexp
	invertMatch bool
	lineNumbers bool
}

func New(pattern string, ignoreCase, invertMatch, lineNumbers bool) (*Searcher, error) {
	if ignoreCase {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &Searcher{
		pattern:     re,
		invertMatch: invertMatch,
		lineNumbers: lineNumbers,
	}, nil
}

func (s *Searcher) SearchLinesInFile(filename string, nodeID, totalNodes int) ([]string, bool, error) {
	log.Printf("SearchLinesInFile: file=%s, nodeID=%d, totalNodes=%d", filename, nodeID, totalNodes)
	file, err := os.Open(filename)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var wg sync.WaitGroup
	type result struct {
		line  string
		match bool
	}
	ch := make(chan result, 100)

	lineIndex := 0
	for scanner.Scan() {
		originalLine := scanner.Text()
		if lineIndex%totalNodes == nodeID {
			wg.Add(1)
			go func(line string, idx int) {
				defer wg.Done()
				matched := s.pattern.MatchString(line)
				if (s.invertMatch && !matched) || (!s.invertMatch && matched) {
					var outLine string
					if s.lineNumbers {
						outLine = strconv.Itoa(idx+1) + ":" + line
					} else {
						outLine = line
					}
					ch <- result{line: outLine, match: true}
				} else {
					ch <- result{line: "", match: false}
				}
			}(originalLine, lineIndex)
		}
		lineIndex++
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	var lines []string
	anyFound := false
	for res := range ch {
		if res.match {
			lines = append(lines, res.line+"\n")
			anyFound = true
		}
	}
	return lines, anyFound, nil
}
