package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSample(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadQuizSampleCRLFNoTrailingBlank(t *testing.T) {
	content := strings.TrimRight(readSample(t), "\n")
	content = strings.ReplaceAll(content, "\n", "\r\n")
	path := writeFile(t, t.TempDir(), "sample.txt", content)

	q, warnings, err := LoadQuiz(path, 5, 6)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if q.File != "sample" {
		t.Errorf("File = %q, want sample", q.File)
	}
	if len(q.Topics) != 5 {
		t.Fatalf("topics = %d, want 5", len(q.Topics))
	}
	if q.Topics[0] != "Témakör1" || q.Topics[4] != "Témakör5" {
		t.Errorf("topics = %q", q.Topics)
	}
	for i, qs := range q.Questions {
		if len(qs) != 6 {
			t.Fatalf("topic %d questions = %d, want 6", i+1, len(qs))
		}
	}
	if got := q.Questions[1][2]; got != "Kérdés23" {
		t.Errorf("Questions[1][2] = %q, want Kérdés23", got)
	}
	if got := q.Questions[4][5]; got != "Kérdés56" {
		t.Errorf("Questions[4][5] = %q, want Kérdés56", got)
	}
}

func TestLoadQuizLF(t *testing.T) {
	path := writeFile(t, t.TempDir(), "sample.txt", readSample(t))
	if _, _, err := LoadQuiz(path, 5, 6); err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
}

func TestLoadQuizNotUTF8(t *testing.T) {
	// CP1250 bytes, same shape as the legacy files.
	content := "T\xe9mak\xf6r1\r\nK\xe9rd\xe9s11\r\n"
	path := writeFile(t, t.TempDir(), "latin2.txt", content)

	_, _, err := LoadQuiz(path, 1, 1)
	if !errors.Is(err, ErrNotUTF8) {
		t.Fatalf("err = %v, want ErrNotUTF8", err)
	}
	if err.Error() != "File not UTF-8. Convert first." {
		t.Errorf("message = %q", err.Error())
	}
}

func TestLoadQuizBOM(t *testing.T) {
	path := writeFile(t, t.TempDir(), "bom.txt", "\ufeff"+readSample(t))
	q, _, err := LoadQuiz(path, 5, 6)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	if q.Topics[0] != "Témakör1" {
		t.Errorf("topic = %q, BOM not stripped", q.Topics[0])
	}
}

func TestLoadQuizTooFewTopics(t *testing.T) {
	content := "T1\nQ1\n\nT2\nQ2\n"
	path := writeFile(t, t.TempDir(), "short.txt", content)

	_, _, err := LoadQuiz(path, 3, 1)
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want *FormatError", err)
	}
	if fe.Line != 7 {
		t.Errorf("line = %d, want 7", fe.Line)
	}
	if !strings.Contains(fe.Msg, "missing topic") {
		t.Errorf("msg = %q", fe.Msg)
	}
}

func TestLoadQuizBlankInsideBlock(t *testing.T) {
	content := "T1\nQ1\n\nQ3\n"
	path := writeFile(t, t.TempDir(), "blank.txt", content)

	_, _, err := LoadQuiz(path, 1, 3)
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want *FormatError", err)
	}
	if fe.Line != 3 {
		t.Errorf("line = %d, want 3", fe.Line)
	}
	if !strings.Contains(fe.Msg, "missing question 2") {
		t.Errorf("msg = %q", fe.Msg)
	}
}

func TestLoadQuizExtraLines(t *testing.T) {
	content := "T1\nQ1\nEXTRA1\nEXTRA2\n\nT2\nQ2\n"
	path := writeFile(t, t.TempDir(), "extra.txt", content)

	q, warnings, err := LoadQuiz(path, 2, 1)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	if len(q.Questions[0]) != 1 || q.Questions[0][0] != "Q1" {
		t.Errorf("questions = %v, want [Q1]", q.Questions[0])
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want 2", warnings)
	}
	for _, w := range warnings {
		if !strings.Contains(w, "extra line ignored") {
			t.Errorf("warning = %q", w)
		}
	}
}

func TestLoadQuizExtraLinesAtEnd(t *testing.T) {
	content := "T1\nQ1\n\nFUZZ\n"
	path := writeFile(t, t.TempDir(), "tail.txt", content)

	_, warnings, err := LoadQuiz(path, 1, 1)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "line 4") {
		t.Errorf("warnings = %v, want one at line 4", warnings)
	}
}

func TestLoadQuizLeadingBlankLines(t *testing.T) {
	content := "\n\nT1\nQ1\n"
	path := writeFile(t, t.TempDir(), "lead.txt", content)

	q, warnings, err := LoadQuiz(path, 1, 1)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	if len(warnings) != 2 {
		t.Errorf("warnings = %v, want 2", warnings)
	}
	if q.Topics[0] != "T1" || q.Questions[0][0] != "Q1" {
		t.Errorf("parsed %q / %q", q.Topics[0], q.Questions[0][0])
	}
}

func TestLoadQuizImages(t *testing.T) {
	dir := t.TempDir()
	content := "T1\nQ11\nQ12\n\nT2\nQ21\nQ22\n"
	path := writeFile(t, dir, "quiz.txt", content)

	for _, name := range []string{
		"quiz_11.jpg",
		"quiz_21.jpeg",
		"quiz_22.png",
		"quiz_lightning.png",
	} {
		writeFile(t, dir, name, "x")
	}

	q, _, err := LoadQuiz(path, 2, 2)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	want := [][]bool{{true, false}, {true, true}}
	wantFiles := [][]string{{"quiz_11.jpg", ""}, {"quiz_21.jpeg", "quiz_22.png"}}
	for i := range want {
		for k := range want[i] {
			if q.HasImage[i][k] != want[i][k] {
				t.Errorf("HasImage[%d][%d] = %v, want %v", i, k, q.HasImage[i][k], want[i][k])
			}
			if q.Images[i][k] != wantFiles[i][k] {
				t.Errorf("Images[%d][%d] = %q, want %q", i, k, q.Images[i][k], wantFiles[i][k])
			}
		}
	}
	if !q.LightningImage {
		t.Error("LightningImage = false, want true")
	}
}

func TestLoadQuizNoImages(t *testing.T) {
	path := writeFile(t, t.TempDir(), "plain.txt", "T1\nQ1\n")
	q, _, err := LoadQuiz(path, 1, 1)
	if err != nil {
		t.Fatalf("LoadQuiz: %v", err)
	}
	if q.HasImage[0][0] || q.LightningImage {
		t.Errorf("HasImage = %v, lightning = %v, want all false", q.HasImage, q.LightningImage)
	}
}

func TestLoadQuizBadSize(t *testing.T) {
	for _, tc := range []struct {
		topics, questions int
	}{
		{0, 6},
		{5, 0},
		{10, 6},
		{5, 10},
	} {
		_, _, err := LoadQuiz("does-not-exist.txt", tc.topics, tc.questions)
		if err == nil || !strings.Contains(err.Error(), "board size") {
			t.Errorf("size %dx%d: err = %v, want board size error", tc.topics, tc.questions, err)
		}
	}
}

func TestLoadQuizEmptyFile(t *testing.T) {
	path := writeFile(t, t.TempDir(), "empty.txt", "")
	_, _, err := LoadQuiz(path, 1, 1)
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want *FormatError", err)
	}
	if fe.Line != 1 || fe.Msg != "empty file" {
		t.Errorf("err = %v", fe)
	}
}

func TestLoadQuizMissingFile(t *testing.T) {
	_, _, err := LoadQuiz(filepath.Join(t.TempDir(), "missing.txt"), 1, 1)
	if err == nil {
		t.Fatal("err = nil, want file error")
	}
	if errors.Is(err, ErrNotUTF8) {
		t.Errorf("err = %v, want os file error", err)
	}
}

func TestQuizPath(t *testing.T) {
	dir := t.TempDir()
	got, err := quizPath(dir, "quiz.txt")
	if err != nil || got != filepath.Join(dir, "quiz.txt") {
		t.Fatalf("quizPath = %q, %v", got, err)
	}
	if _, err := quizPath(dir, "QUIZ.TXT"); err != nil {
		t.Errorf("uppercase extension rejected: %v", err)
	}
	for _, name := range []string{"", "../quiz.txt", `..\quiz.txt`, "sub/quiz.txt", "quiz.jpg", "."} {
		if _, err := quizPath(dir, name); !errors.Is(err, ErrBadQuizName) {
			t.Errorf("quizPath(%q) = %v, want ErrBadQuizName", name, err)
		}
	}
}

func TestListQuizzes(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.txt", "A.TXT", "note.md"} {
		writeFile(t, dir, name, "x")
	}
	if err := os.Mkdir(filepath.Join(dir, "folder.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	names, err := listQuizzes(dir)
	if err != nil {
		t.Fatalf("listQuizzes: %v", err)
	}
	want := []string{"A.TXT", "b.txt"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}

	missing, err := listQuizzes(filepath.Join(dir, "nope"))
	if err != nil || len(missing) != 0 {
		t.Fatalf("missing dir = %v, %v; want empty", missing, err)
	}
}
