package schemaaudit

import (
	"regexp"
	"sort"
	"strings"
)

var (
	tableRegex      = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?"?([a-zA-Z0-9_\.]+)"?`)
	indexRegex      = regexp.MustCompile(`(?is)CREATE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?"?([a-zA-Z0-9_\.]+)"?`)
	constraintRegex = regexp.MustCompile(`(?is)ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?"?([a-zA-Z0-9_\.]+)"?.*?ADD\s+CONSTRAINT\s+"?([a-zA-Z0-9_\.]+)"?`)
)

// Diff captures differences between master schema and tenant template.
type Diff struct {
	MissingTables      []string
	MissingIndexes     []string
	MissingConstraints []string
}

// HasDifferences returns true if any difference is present.
func (d Diff) HasDifferences() bool {
	return len(d.MissingTables) > 0 || len(d.MissingIndexes) > 0 || len(d.MissingConstraints) > 0
}

// Format renders the diff in a human readable form suitable for logs/reports.
func (d Diff) Format() string {
	if !d.HasDifferences() {
		return "No obvious differences detected between master and tenant template schemas."
	}

	var b strings.Builder
	if len(d.MissingTables) > 0 {
		b.WriteString("Tables missing from tenant template (present in master):\n")
		for _, t := range d.MissingTables {
			b.WriteString("  - " + t + "\n")
		}
		b.WriteString("\n")
	}

	if len(d.MissingIndexes) > 0 {
		b.WriteString("Indexes missing from tenant template:\n")
		for _, idx := range d.MissingIndexes {
			b.WriteString("  - " + idx + "\n")
		}
		b.WriteString("\n")
	}

	if len(d.MissingConstraints) > 0 {
		b.WriteString("Constraints missing from tenant template:\n")
		for _, c := range d.MissingConstraints {
			b.WriteString("  - " + c + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Review the differences above and update the tenant template accordingly.")
	return b.String()
}

// Analyze compares master and tenant template schema SQL definitions.
func Analyze(master, template string) Diff {
	masterTables := extractMatches(master, tableRegex, 1)
	templateTables := extractMatches(template, tableRegex, 1)
	masterIndexes := extractMatches(master, indexRegex, 1)
	templateIndexes := extractMatches(template, indexRegex, 1)
	masterConstraints := extractConstraintNames(master)
	templateConstraints := extractConstraintNames(template)

	return Diff{
		MissingTables:      difference(masterTables, templateTables),
		MissingIndexes:     difference(masterIndexes, templateIndexes),
		MissingConstraints: difference(masterConstraints, templateConstraints),
	}
}

func extractMatches(input string, re *regexp.Regexp, group int) []string {
	matches := re.FindAllStringSubmatch(input, -1)
	set := map[string]struct{}{}
	for _, m := range matches {
		if len(m) > group {
			set[normaliseIdentifier(m[group])] = struct{}{}
		}
	}
	return setToSortedSlice(set)
}

func extractConstraintNames(input string) []string {
	matches := constraintRegex.FindAllStringSubmatch(input, -1)
	set := map[string]struct{}{}
	for _, m := range matches {
		if len(m) > 2 {
			set[normaliseIdentifier(m[2])] = struct{}{}
		}
	}
	return setToSortedSlice(set)
}

func difference(master, template []string) []string {
	tmplSet := map[string]struct{}{}
	for _, v := range template {
		tmplSet[v] = struct{}{}
	}

	var diff []string
	for _, v := range master {
		if _, exists := tmplSet[v]; !exists {
			diff = append(diff, v)
		}
	}
	return diff
}

func setToSortedSlice(set map[string]struct{}) []string {
	slice := make([]string, 0, len(set))
	for k := range set {
		slice = append(slice, k)
	}
	sort.Strings(slice)
	return slice
}

func normaliseIdentifier(id string) string {
	id = strings.TrimSpace(id)
	id = strings.Trim(id, "\"")
	return strings.ToLower(id)
}
