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

type line struct {
	Test    string `json:"Test"`
	Package string `json:"Package"`
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Printf("Error occured: %v\n", err)
		os.Exit(1)
	}
}

func run(input io.Reader, output io.Writer) error {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	dataPerLine := strings.Split(string(data), "\n")

	testsByPackage := map[string][]string{}

	for _, lineRaw := range dataPerLine {
		line := &line{}

		if lineRaw == "" {
			continue
		}

		if err := json.Unmarshal([]byte(lineRaw), line); err != nil {
			return fmt.Errorf("decoding line %q: %w", lineRaw, err)
		}

		if line.Test == "" {
			continue
		}

		testsByPackage[line.Package] = append(testsByPackage[line.Package], line.Test)
	}

	for p, tests := range testsByPackage {
		uniqueTests := []string{}

		testNames := map[string]struct{}{}

		for _, test := range tests {
			if _, ok := testNames[test]; !ok {
				uniqueTests = append(uniqueTests, strings.ReplaceAll(strings.TrimPrefix(test, "Test_"), "_", " "))
				testNames[test] = struct{}{}
			}
		}

		sort.Strings(uniqueTests)

		fmt.Fprintf(output, "%s:\n", p)

		if err := drill(uniqueTests, "  ", output); err != nil {
			return fmt.Errorf("generating output for package %q: %w", p, err)
		}
	}

	return nil
}

func drill(tests []string, prefix string, output io.Writer) error {
	red := color.New(color.FgHiBlack)

	for _, test := range topLevelTests(tests) {
		st := subTests(tests, test)

		if _, err := fmt.Fprintf(output, "%s%s\n", red.Sprintf(prefix), test); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}

		if len(st) == 0 {
			continue
		}

		if err := drill(st, prefix+test+" ", output); err != nil {
			return fmt.Errorf("generating output for subtests: %w", err)
		}
	}

	return nil
}

func topLevelTests(tests []string) []string {
	topLevelTests := []string{}

	for _, test := range tests {
		if !strings.Contains(test, "/") {
			topLevelTests = append(topLevelTests, test)
		}
	}

	return topLevelTests
}

func subTests(tests []string, parentName string) []string {
	subTests := []string{}

	for _, test := range tests {
		if strings.HasPrefix(test, parentName) && test != parentName {
			subTests = append(subTests, strings.TrimPrefix(test, parentName+"/"))
		}
	}

	return subTests
}
