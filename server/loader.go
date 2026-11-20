package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// ErrNotUTF8 is plan D12: no silent encoding guessing, bad bytes are an error.
var ErrNotUTF8 = errors.New("File not UTF-8. Convert first.")

// ErrBadQuizName rejects anything that is not a plain .txt file name.
var ErrBadQuizName = errors.New("bad quiz file name")

// FormatError reports a parse problem at a 1-based line number.
type FormatError struct {
	Line int
	Msg  string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

// Quiz is a parsed legacy question file.
// Questions, HasImage and Images are indexed [topic][question].
type Quiz struct {
	File           string // base name without extension, used for image lookup
	Topics         []string
	Questions      [][]string
	HasImage       [][]bool
	Images         [][]string // file name with extension, empty = no image
	LightningImage string     // file name with extension, empty = no image
}

const (
	minBoardSize = 1
	maxTopics    = 7
	maxQuestions = 6
)

var imageExts = []string{".jpg", ".jpeg", ".png"}

// LoadQuiz reads a legacy .txt quiz file. Board size comes from settings (plan D19).
// It returns warnings for ignored extra lines and a *FormatError for broken structure.
func LoadQuiz(path string, topics, questions int) (*Quiz, []string, error) {
	if topics < minBoardSize || topics > maxTopics ||
		questions < minBoardSize || questions > maxQuestions {
		return nil, nil, fmt.Errorf("board size must be topics %d-%d, questions %d-%d, got %dx%d",
			minBoardSize, maxTopics, minBoardSize, maxQuestions, topics, questions)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	if !utf8.Valid(raw) {
		return nil, nil, ErrNotUTF8
	}

	lines := splitLines(string(raw))
	if len(lines) == 0 {
		return nil, nil, &FormatError{Line: 1, Msg: "empty file"}
	}

	q := &Quiz{File: quizBase(path)}
	var warnings []string
	i := 0

	// Leading blank lines are not part of any block.
	for i < len(lines) && lines[i] == "" {
		warnings = append(warnings, fmt.Sprintf("line %d: blank line ignored", i+1))
		i++
	}

	for t := 0; t < topics; t++ {
		if i >= len(lines) {
			return nil, warnings, &FormatError{
				Line: len(lines) + 1,
				Msg:  fmt.Sprintf("missing topic line %d", t+1),
			}
		}
		topic := lines[i]
		i++

		qs := make([]string, 0, questions)
		for k := 0; k < questions; k++ {
			if i >= len(lines) {
				return nil, warnings, &FormatError{
					Line: len(lines) + 1,
					Msg:  fmt.Sprintf("missing question %d for topic %q", k+1, topic),
				}
			}
			if lines[i] == "" {
				return nil, warnings, &FormatError{
					Line: i + 1,
					Msg:  fmt.Sprintf("missing question %d for topic %q", k+1, topic),
				}
			}
			qs = append(qs, lines[i])
			i++
		}

		// Extra non-blank lines before the blank separator: keep the first N
		// questions, warn about the rest (plan: extra lines ignored).
		for i < len(lines) && lines[i] != "" {
			warnings = append(warnings, fmt.Sprintf("line %d: extra line ignored", i+1))
			i++
		}
		// Blank separator, plus any extra blank lines.
		for i < len(lines) && lines[i] == "" {
			i++
		}

		q.Topics = append(q.Topics, topic)
		q.Questions = append(q.Questions, qs)
	}

	for ; i < len(lines); i++ {
		if lines[i] != "" {
			warnings = append(warnings, fmt.Sprintf("line %d: extra line ignored", i+1))
		}
	}

	q.findImages(path, questions)
	return q, warnings, nil
}

// splitLines normalizes CRLF/CR to LF, strips a UTF-8 BOM and trailing
// spaces/tabs, then splits. Empty input yields no lines.
func splitLines(s string) []string {
	s = strings.TrimPrefix(s, "\ufeff")
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return lines
}

func quizBase(path string) string {
	name := filepath.Base(path)
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// findImages sets HasImage, Images and LightningImage from files next to the
// quiz: <base>_<topic><question>.<ext> and <base>_lightning.<ext>. Topic and
// question numbers are 1-based, max 7 and 6, so one digit each (plan D19).
func (q *Quiz) findImages(path string, questions int) {
	dir := filepath.Dir(path)
	names := dirFileNames(dir)
	q.HasImage = make([][]bool, len(q.Topics))
	q.Images = make([][]string, len(q.Topics))
	for t := range q.Topics {
		q.HasImage[t] = make([]bool, questions)
		q.Images[t] = make([]string, questions)
		for k := 0; k < questions; k++ {
			name := fmt.Sprintf("%s_%d%d", q.File, t+1, k+1)
			q.Images[t][k] = findImage(names, name)
			q.HasImage[t][k] = q.Images[t][k] != ""
		}
	}
	q.LightningImage = findImage(names, q.File+"_lightning")
}

// dirFileNames lists regular files by their exact stored names. Comparing
// names from ReadDir, not os.Stat, keeps matching case-sensitive on Windows
// and macOS, where the filesystem would otherwise accept a different case.
func dirFileNames(dir string) map[string]bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Type().IsRegular() {
			names[e.Name()] = true
		}
	}
	return names
}

// findImage returns the file name with its extension, or "" when missing.
// The name must match exactly: no case folding and no Unicode normalization.
func findImage(names map[string]bool, name string) string {
	for _, ext := range imageExts {
		if names[name+ext] {
			return name + ext
		}
	}
	return ""
}

// quizPath resolves a quiz file name against dir. Only plain .txt names are
// accepted, so admin messages cannot escape the quizzes folder.
func quizPath(dir, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\`) || name != filepath.Base(name) {
		return "", ErrBadQuizName
	}
	if !strings.EqualFold(filepath.Ext(name), ".txt") {
		return "", ErrBadQuizName
	}
	return filepath.Join(dir, name), nil
}

// listQuizzes returns sorted .txt file names in dir. Missing dir = empty list.
func listQuizzes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".txt") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
