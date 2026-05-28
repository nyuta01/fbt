package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Result struct {
	LeftName  string          `json:"left_name"`
	RightName string          `json:"right_name"`
	Sections  []SectionChange `json:"sections,omitempty"`
	Unified   string          `json:"unified"`
}

type SectionChange struct {
	Heading string `json:"heading"`
	Status  string `json:"status"`
}

func ComparePaths(leftPath, rightPath, leftName, rightName string) (Result, error) {
	left, err := readText(leftPath)
	if err != nil {
		return Result{}, err
	}
	right, err := readText(rightPath)
	if err != nil {
		return Result{}, err
	}
	return CompareText(left, right, leftName, rightName), nil
}

func CompareText(left, right, leftName, rightName string) Result {
	return Result{
		LeftName:  leftName,
		RightName: rightName,
		Sections:  sectionChanges(left, right),
		Unified:   unifiedDiff(left, right, leftName, rightName),
	}
}

func readText(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		return string(data), err
	}
	index := filepath.Join(path, "index.md")
	if _, err := os.Stat(index); err == nil {
		data, err := os.ReadFile(index)
		return string(data), err
	}
	var files []string
	if err := filepath.WalkDir(path, func(entryPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, entryPath)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	var builder strings.Builder
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		builder.WriteString("## ")
		builder.WriteString(filepath.ToSlash(file))
		builder.WriteString("\n\n")
		builder.Write(data)
		if !strings.HasSuffix(builder.String(), "\n") {
			builder.WriteByte('\n')
		}
	}
	return builder.String(), nil
}

func sectionChanges(left, right string) []SectionChange {
	leftSections := markdownSections(left)
	rightSections := markdownSections(right)
	seen := map[string]struct{}{}
	var headings []string
	for heading := range leftSections {
		seen[heading] = struct{}{}
		headings = append(headings, heading)
	}
	for heading := range rightSections {
		if _, ok := seen[heading]; !ok {
			headings = append(headings, heading)
		}
	}
	sort.Strings(headings)
	var changes []SectionChange
	for _, heading := range headings {
		leftHash, leftOK := leftSections[heading]
		rightHash, rightOK := rightSections[heading]
		switch {
		case !leftOK:
			changes = append(changes, SectionChange{Heading: heading, Status: "added"})
		case !rightOK:
			changes = append(changes, SectionChange{Heading: heading, Status: "removed"})
		case leftHash != rightHash:
			changes = append(changes, SectionChange{Heading: heading, Status: "changed"})
		}
	}
	return changes
}

func markdownSections(content string) map[string]string {
	sections := map[string]string{}
	current := "(root)"
	var builder strings.Builder
	flush := func() {
		sum := sha256.Sum256([]byte(builder.String()))
		sections[current] = hex.EncodeToString(sum[:])
		builder.Reset()
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			flush()
			current = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if current == "" {
				current = "(unnamed)"
			}
			continue
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	flush()
	return sections
}

func unifiedDiff(left, right, leftName, rightName string) string {
	leftLines := splitLines(left)
	rightLines := splitLines(right)
	table := lcsTable(leftLines, rightLines)
	var lines []string
	lines = append(lines, fmt.Sprintf("--- %s", leftName), fmt.Sprintf("+++ %s", rightName))
	buildDiff(&lines, table, leftLines, rightLines, len(leftLines), len(rightLines))
	if len(lines) == 2 {
		lines = append(lines, " no differences")
	}
	return strings.Join(lines, "\n") + "\n"
}

func splitLines(value string) []string {
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func lcsTable(left, right []string) [][]int {
	table := make([][]int, len(left)+1)
	for i := range table {
		table[i] = make([]int, len(right)+1)
	}
	for i := len(left) - 1; i >= 0; i-- {
		for j := len(right) - 1; j >= 0; j-- {
			if left[i] == right[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else if table[i+1][j] >= table[i][j+1] {
				table[i][j] = table[i+1][j]
			} else {
				table[i][j] = table[i][j+1]
			}
		}
	}
	return table
}

func buildDiff(out *[]string, table [][]int, left, right []string, i, j int) {
	if i > 0 && j > 0 && left[i-1] == right[j-1] {
		buildDiff(out, table, left, right, i-1, j-1)
		*out = append(*out, " "+left[i-1])
		return
	}
	if j > 0 && (i == 0 || table[i][j-1] >= table[i-1][j]) {
		buildDiff(out, table, left, right, i, j-1)
		*out = append(*out, "+"+right[j-1])
		return
	}
	if i > 0 && (j == 0 || table[i][j-1] < table[i-1][j]) {
		buildDiff(out, table, left, right, i-1, j)
		*out = append(*out, "-"+left[i-1])
	}
}
