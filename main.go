package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// Copy of https://pkg.go.dev/cmd/test2json#hdr-Output_Format format.
type testOutputLine struct {
	Test    string `json:"Test"`
	Package string `json:"Package"`
	Action  string `json:"Action"`
}

type testTree struct {
	name       string
	action     string
	subTests   []*testTree
	parentName string
}

const (
	failAction = "fail"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error occurred: %v\n", err)
		os.Exit(1)
	}
}

func outputToLines(input io.Reader) ([]*testOutputLine, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	dataPerLine := strings.Split(string(data), "\n")

	lines := []*testOutputLine{}

	for _, lineRaw := range dataPerLine {
		line := &testOutputLine{}

		if lineRaw == "" {
			continue
		}

		if err := json.Unmarshal([]byte(lineRaw), line); err != nil {
			return nil, fmt.Errorf("decoding line %q: %w", lineRaw, err)
		}

		if line.Test == "" {
			continue
		}

		lines = append(lines, line)
	}

	return lines, nil
}

func getFinalLines(lines []*testOutputLine) []*testOutputLine {
	finalLines := []*testOutputLine{}

	for _, line := range lines {
		if line.Action == "pass" || line.Action == failAction {
			finalLines = append(finalLines, line)
		}
	}

	return finalLines
}

func formatName(name string) string {
	nameWithoutPrefixAndUnderscores := strings.ReplaceAll(strings.TrimPrefix(name, "Test_"), "_", " ")

	if strings.Contains(nameWithoutPrefixAndUnderscores, "/") {
		return strings.TrimSpace(strings.ReplaceAll(nameWithoutPrefixAndUnderscores, "/", " "))
	}

	return strings.TrimSpace(nameWithoutPrefixAndUnderscores)
}

func formatTestTree(trees []*testTree, parentName string) []string {
	result := []string{}

	for _, test := range trees {
		output := ""

		normalizedName := formatName(test.name)

		dimmedWhite := color.New(color.FgHiBlack)

		childName := strings.TrimPrefix(normalizedName, parentName)

		switch {
		case parentName == "" && test.action == failAction:
			output += color.New(color.FgRed).Sprintf(normalizedName)
		case parentName == "" && test.action != failAction:
			output += normalizedName
		case parentName != "" && test.action == failAction:
			output += dimmedWhite.Sprintf(parentName)
			output += color.New(color.FgHiRed).Sprintf(childName)
		case parentName != "" && test.action != failAction:
			output += dimmedWhite.Sprintf(parentName)
			output += childName
		}

		result = append(result, output)
		result = append(result, formatTestTree(test.subTests, normalizedName)...)
	}

	return result
}

func linesToTestTrees(lines []*testOutputLine, parentKeys []string) []*testTree {
	result := []*testTree{}

	for _, line := range lines {
		splitted := strings.Split(line.Test, "/")

		// Consider only child items, not grand-children etc.
		if len(splitted) != len(parentKeys)+1 || !strings.HasPrefix(line.Test, strings.Join(parentKeys, "/")) {
			continue
		}

		result = append(result, &testTree{
			name:       line.Test,
			action:     line.Action,
			parentName: strings.Join(parentKeys, "/"),
			subTests:   linesToTestTrees(lines, splitted),
		})
	}

	// Put failed test cases on top.
	sort.Slice(result, func(i, j int) bool {
		return result[i].action == failAction
	})

	return result
}

func groupLinesPerPackage(lines []*testOutputLine) (map[string][]*testOutputLine, []string) {
	packages := []string{}

	linesByPackage := map[string][]*testOutputLine{}

	for _, lineRaw := range getFinalLines(lines) {
		if _, ok := linesByPackage[lineRaw.Package]; !ok {
			packages = append(packages, lineRaw.Package)
		}

		linesByPackage[lineRaw.Package] = append(linesByPackage[lineRaw.Package], lineRaw)
	}

	sort.Strings(packages)

	return linesByPackage, packages
}

func run(input io.Reader, output io.Writer) error {
	lines, err := outputToLines(input)
	if err != nil {
		return fmt.Errorf("converting input to lines format: %w", err)
	}

	linesByPackage, packages := groupLinesPerPackage(getFinalLines(lines))

	for _, p := range packages {
		fmt.Fprintf(output, "%s:\n", p)

		lines := formatTestTree(linesToTestTrees(linesByPackage[p], []string{}), "")

		for _, line := range lines {
			fmt.Fprintf(output, "  %s\n", line)
		}

		fmt.Fprintln(output)
	}

	return nil
}
