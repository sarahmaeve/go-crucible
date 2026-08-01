// Command verify checks the structural invariants that protect the
// crucible against exercise rot. It is the fast half of the local
// verification harness (`make verify-quick`); the slow half — running
// tests, vetting, and round-tripping solution patches — lives in the
// Makefile (`make verify`).
//
// Checks:
//
//  1. Registry <-> tree consistency: every numbered exercise in
//     .crucible/exercises.yaml has its directory, docs, patch, target
//     file, and a TestExerciseNN function — and every exercise
//     directory and solution patch in the tree is registered.
//  2. Review, diagnosis, and Wheel exercises have their full file sets, and
//     their draws_on/localizes fields point at real numbered exercises.
//  3. Diagnosis artifact pins: every {file, line, contains} pin in the
//     registry matches the tree, and every go-crucible file:line
//     mentioned inside an ARTIFACT.txt names a file that exists and is
//     long enough.
//  4. Spoiler lint: comments under internal/ and cmd/ must not reveal
//     exact diagnoses or fixes. Production comments also reject exercise
//     numbers and planted-spoiler markers.
//  5. Makefile drift: the EXERCISES and PRESOLVED lists match the
//     registry's exercise numbers and solved_in_main fields.
//  6. Formatting: every Go file under cmd/, internal/, and tools/ matches
//     gofmt output.
//
// Run from the repository root: go run ./tools/verify
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type registry struct {
	Exercises          []exercise          `yaml:"exercises"`
	ReviewExercises    []reviewExercise    `yaml:"review_exercises"`
	WheelExercises     []wheelExercise     `yaml:"wheel_exercises"`
	DiagnosisExercises []diagnosisExercise `yaml:"diagnosis_exercises"`
}

type exercise struct {
	Number       string `yaml:"number"`
	Title        string `yaml:"title"`
	File         string `yaml:"file"`
	Patch        string `yaml:"patch"`
	SolvedInMain string `yaml:"solved_in_main"`
}

type reviewExercise struct {
	Number    string   `yaml:"number"`
	Directory string   `yaml:"directory"`
	DrawsOn   []string `yaml:"draws_on"`
}

type wheelExercise struct {
	Number           string   `yaml:"number"`
	Directory        string   `yaml:"directory"`
	DrawsOn          []string `yaml:"draws_on"`
	Report           string   `yaml:"report"`
	Candidate        string   `yaml:"candidate"`
	Evidence         []string `yaml:"evidence"`
	Facilitator      string   `yaml:"facilitator"`
	Debrief          string   `yaml:"debrief"`
	ProductionEscape string   `yaml:"production_escape"`
}

type pin struct {
	Line     int    `yaml:"line"`
	Contains string `yaml:"contains"`
}

type reference struct {
	File string `yaml:"file"`
	Pins []pin  `yaml:"pins"`
}

type diagnosisExercise struct {
	Number     string      `yaml:"number"`
	Directory  string      `yaml:"directory"`
	Artifact   string      `yaml:"artifact"`
	Localizes  string      `yaml:"localizes"`
	References []reference `yaml:"references"`
}

var numberedDir = regexp.MustCompile(`^\d{2}-`)

func main() {
	regPath := filepath.Join(".crucible", "exercises.yaml")
	data, err := os.ReadFile(regPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify: cannot read %s — run from the repository root (%v)\n", regPath, err)
		os.Exit(2)
	}
	var reg registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		fmt.Fprintf(os.Stderr, "verify: parsing %s: %v\n", regPath, err)
		os.Exit(2)
	}

	total := 0
	run := func(name string, fn func() []string) {
		ps := fn()
		if len(ps) == 0 {
			fmt.Printf("ok:   %s\n", name)
			return
		}
		fmt.Printf("FAIL: %s\n", name)
		for _, p := range ps {
			fmt.Printf("      - %s\n", p)
		}
		total += len(ps)
	}

	run(fmt.Sprintf("numbered exercises (%d)", len(reg.Exercises)),
		func() []string { return checkNumbered(reg.Exercises) })
	run(fmt.Sprintf("review exercises (%d)", len(reg.ReviewExercises)),
		func() []string { return checkReview(reg) })
	run(fmt.Sprintf("Wheel exercises (%d)", len(reg.WheelExercises)),
		func() []string { return checkWheel(reg) })
	run(fmt.Sprintf("diagnosis exercises (%d)", len(reg.DiagnosisExercises)),
		func() []string { return checkDiagnosis(reg) })
	run("artifact file:line references",
		func() []string { return checkArtifactScan(reg.DiagnosisExercises) })
	run("spoiler lint (internal/, cmd/)", checkSpoilers)
	run("Makefile EXERCISES/PRESOLVED drift",
		func() []string { return checkMakefile(reg.Exercises) })
	run("gofmt (cmd/, internal/, tools/)", checkFormatting)

	if total > 0 {
		fmt.Printf("\nverify-quick: %d problem(s)\n", total)
		os.Exit(1)
	}
	fmt.Println("\nverify-quick: all structural checks passed")
}

func checkNumbered(exercises []exercise) []string {
	var ps []string
	tests := testFunctions()

	slugs := map[string]bool{}
	patches := map[string]bool{}
	for _, e := range exercises {
		if e.Number == "" || e.Patch == "" || e.File == "" {
			ps = append(ps, fmt.Sprintf("exercise %q: number, file, and patch are all required", e.Number))
			continue
		}
		slug := strings.TrimSuffix(filepath.Base(e.Patch), ".patch")
		slugs[slug] = true
		patches[filepath.Base(e.Patch)] = true

		if !fileExists(e.Patch) {
			ps = append(ps, fmt.Sprintf("exercise %s: patch %s missing", e.Number, e.Patch))
		} else if data, err := os.ReadFile(e.Patch); err == nil && !strings.Contains(string(data), e.File) {
			ps = append(ps, fmt.Sprintf("exercise %s: patch %s never mentions its target file %s", e.Number, e.Patch, e.File))
		}
		if !fileExists(e.File) {
			ps = append(ps, fmt.Sprintf("exercise %s: target file %s missing", e.Number, e.File))
		}
		dir := filepath.Join("exercises", slug)
		for _, f := range []string{"README.md", "HINTS.md"} {
			if !fileExists(filepath.Join(dir, f)) {
				ps = append(ps, fmt.Sprintf("exercise %s: %s missing", e.Number, filepath.Join(dir, f)))
			}
		}
		if !tests[e.Number] {
			ps = append(ps, fmt.Sprintf("exercise %s: no TestExercise%s function found under internal/ or cmd/", e.Number, e.Number))
		}
	}

	for _, ent := range readDir("exercises") {
		if ent.IsDir() && numberedDir.MatchString(ent.Name()) && !slugs[ent.Name()] {
			ps = append(ps, fmt.Sprintf("exercises/%s/ has no registry entry", ent.Name()))
		}
	}
	for _, ent := range readDir("solutions") {
		if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".patch") && !patches[ent.Name()] {
			ps = append(ps, fmt.Sprintf("solutions/%s has no registry entry", ent.Name()))
		}
	}
	return ps
}

// testFunctions scans test files for TestExerciseNN functions and
// returns the set of two-digit exercise numbers that have one.
func testFunctions() map[string]bool {
	found := map[string]bool{}
	re := regexp.MustCompile(`func TestExercise(\d{2})`)
	for _, root := range []string{"internal", "cmd"} {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, m := range re.FindAllStringSubmatch(string(data), -1) {
				found[m[1]] = true
			}
			return nil
		})
	}
	return found
}

func checkReview(reg registry) []string {
	var ps []string
	numbered := numberSet(reg.Exercises)
	dirs := map[string]bool{}
	for _, r := range reg.ReviewExercises {
		dir := strings.TrimSuffix(r.Directory, "/")
		dirs[filepath.Base(dir)] = true
		for _, f := range []string{"README.md", "HINTS.md", "REVIEW_TEMPLATE.md", "REVIEWER_NOTES.md"} {
			if !fileExists(filepath.Join(dir, f)) {
				ps = append(ps, fmt.Sprintf("review %s: %s missing", r.Number, filepath.Join(dir, f)))
			}
		}
		for _, n := range r.DrawsOn {
			if !numbered[n] {
				ps = append(ps, fmt.Sprintf("review %s: draws_on %q is not a numbered exercise", r.Number, n))
			}
		}
	}
	for _, ent := range readDir(filepath.Join("exercises", "review")) {
		if ent.IsDir() && numberedDir.MatchString(ent.Name()) && !dirs[ent.Name()] {
			ps = append(ps, fmt.Sprintf("exercises/review/%s/ has no registry entry", ent.Name()))
		}
	}
	return ps
}

func checkWheel(reg registry) []string {
	var ps []string
	numbered := numberSet(reg.Exercises)
	dirs := map[string]bool{}

	for _, path := range []string{
		filepath.Join("exercises", "wheel", "README.md"),
		filepath.Join("exercises", "wheel", "WORKSHEET.template.md"),
		filepath.Join("exercises", "wheel", "RUBRIC.md"),
	} {
		if !fileExists(path) {
			ps = append(ps, fmt.Sprintf("Wheel track: %s missing", path))
		}
	}

	for _, w := range reg.WheelExercises {
		dir := strings.TrimSuffix(w.Directory, "/")
		dirs[filepath.Base(dir)] = true
		if w.Number == "" || w.Directory == "" {
			ps = append(ps, fmt.Sprintf("Wheel %q: number and directory are required", w.Number))
		}
		if strings.TrimSpace(w.ProductionEscape) == "" {
			ps = append(ps, fmt.Sprintf("Wheel %s: production_escape is required", w.Number))
		}
		for _, path := range []string{
			w.Report,
			w.Candidate,
			filepath.Join(dir, "evidence", "README.md"),
			w.Facilitator,
			w.Debrief,
		} {
			if path == "" || !fileExists(path) {
				ps = append(ps, fmt.Sprintf("Wheel %s: %s missing", w.Number, path))
			}
		}
		if len(w.Evidence) == 0 {
			ps = append(ps, fmt.Sprintf("Wheel %s: no evidence packets registered", w.Number))
		}
		for _, path := range w.Evidence {
			if !fileExists(path) {
				ps = append(ps, fmt.Sprintf("Wheel %s: evidence packet %s missing", w.Number, path))
			}
		}
		for _, n := range w.DrawsOn {
			if !numbered[n] {
				ps = append(ps, fmt.Sprintf("Wheel %s: draws_on %q is not a numbered exercise", w.Number, n))
			}
		}
		if !pathWithin(w.Report, dir) || !pathWithin(w.Candidate, dir) {
			ps = append(ps, fmt.Sprintf("Wheel %s: report and candidate guide must stay under %s", w.Number, dir))
		}
		for _, path := range w.Evidence {
			if !pathWithin(path, filepath.Join(dir, "evidence")) {
				ps = append(ps, fmt.Sprintf("Wheel %s: evidence packet %s is outside the candidate evidence directory", w.Number, path))
			}
		}
		if !pathWithin(w.Facilitator, filepath.Join(".crucible", "wheel")) {
			ps = append(ps, fmt.Sprintf("Wheel %s: facilitator truth must live under .crucible/wheel", w.Number))
		}
		if !pathWithin(w.Debrief, filepath.Join("solutions", "wheel")) {
			ps = append(ps, fmt.Sprintf("Wheel %s: debrief must live under solutions/wheel", w.Number))
		}
	}

	for _, ent := range readDir(filepath.Join("exercises", "wheel")) {
		if ent.IsDir() && numberedDir.MatchString(ent.Name()) && !dirs[ent.Name()] {
			ps = append(ps, fmt.Sprintf("exercises/wheel/%s/ has no registry entry", ent.Name()))
		}
	}
	return ps
}

func checkDiagnosis(reg registry) []string {
	var ps []string
	numbered := numberSet(reg.Exercises)
	dirs := map[string]bool{}
	for _, d := range reg.DiagnosisExercises {
		dir := strings.TrimSuffix(d.Directory, "/")
		dirs[filepath.Base(dir)] = true
		for _, f := range []string{"README.md", "ARTIFACT.txt", "DIAGNOSIS_TEMPLATE.md", "HINTS.md", "DIAGNOSIS_NOTES.md"} {
			if !fileExists(filepath.Join(dir, f)) {
				ps = append(ps, fmt.Sprintf("diagnosis %s: %s missing", d.Number, filepath.Join(dir, f)))
			}
		}
		if d.Artifact != "" && !fileExists(d.Artifact) {
			ps = append(ps, fmt.Sprintf("diagnosis %s: artifact %s missing", d.Number, d.Artifact))
		}
		if !numbered[d.Localizes] {
			ps = append(ps, fmt.Sprintf("diagnosis %s: localizes %q is not a numbered exercise", d.Number, d.Localizes))
		}
		if len(d.References) == 0 {
			ps = append(ps, fmt.Sprintf("diagnosis %s: no references with pins — the artifact is unprotected against line drift", d.Number))
		}
		for _, ref := range d.References {
			lines, err := fileLines(ref.File)
			if err != nil {
				ps = append(ps, fmt.Sprintf("diagnosis %s: referenced file %s unreadable: %v", d.Number, ref.File, err))
				continue
			}
			for _, p := range ref.Pins {
				if p.Line < 1 || p.Line > len(lines) {
					ps = append(ps, fmt.Sprintf("diagnosis %s: pin %s:%d out of range (file has %d lines)", d.Number, ref.File, p.Line, len(lines)))
					continue
				}
				if !strings.Contains(lines[p.Line-1], p.Contains) {
					ps = append(ps, fmt.Sprintf("diagnosis %s: %s:%d does not contain %q (line is %q) — artifact has rotted, fix artifact + pins together",
						d.Number, ref.File, p.Line, p.Contains, strings.TrimSpace(lines[p.Line-1])))
				}
			}
		}
	}
	for _, ent := range readDir(filepath.Join("exercises", "diagnosis")) {
		if ent.IsDir() && numberedDir.MatchString(ent.Name()) && !dirs[ent.Name()] {
			ps = append(ps, fmt.Sprintf("exercises/diagnosis/%s/ has no registry entry", ent.Name()))
		}
	}
	return ps
}

// artifactRef matches in-repo file:line references inside artifacts,
// e.g. "/build/go-crucible/internal/ingest/reader.go:18". The
// go-crucible/ anchor keeps stdlib paths like
// /usr/local/go/src/internal/poll/... from matching.
var artifactRef = regexp.MustCompile(`go-crucible/((?:internal|cmd)/[A-Za-z0-9_./-]+\.go):(\d+)`)

// checkArtifactScan is the zero-config backstop to the explicit pins:
// every in-repo file:line an artifact mentions must at least name an
// existing file with that many lines.
func checkArtifactScan(diags []diagnosisExercise) []string {
	var ps []string
	for _, d := range diags {
		if d.Artifact == "" {
			continue
		}
		data, err := os.ReadFile(d.Artifact)
		if err != nil {
			continue // already reported by checkDiagnosis
		}
		for _, m := range artifactRef.FindAllStringSubmatch(string(data), -1) {
			file := m[1]
			n, _ := strconv.Atoi(m[2])
			lines, err := fileLines(file)
			if err != nil {
				ps = append(ps, fmt.Sprintf("%s references %s, which is unreadable: %v", d.Artifact, file, err))
				continue
			}
			if n < 1 || n > len(lines) {
				ps = append(ps, fmt.Sprintf("%s references %s:%d but the file has only %d lines", d.Artifact, file, n, len(lines)))
			}
		}
	}
	return ps
}

var commonSpoilerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(BUG|FIXME|TODO|XXX|HACK)\b`), // case-sensitive markers
	regexp.MustCompile(`\b(BUGGY|FIXED)\b`),
	regexp.MustCompile(`(?i)\bFAILS?\s+because\b`),
	regexp.MustCompile(`(?i)\b(the\s+fix\s+is|fix\s+is|replace\s+.+\s+with|change\s+.+\s+to)\b`),
	regexp.MustCompile(`(?i)\bprefer\s+\[?errors\.Is\]?\s+over\s+string\s+comparison\b`),
	regexp.MustCompile(`(?i)\bwrap(?:ped|ping|s)?\b[^\n]*%w\b`),
	regexp.MustCompile(`(?i)\bblocked\s+trying\s+to\s+(?:send|receive)\b`),
	regexp.MustCompile(`(?i)\baccumulat\w*\b[^\n]*\buntil\b[^\n]*\b(?:function|returns?)\b`),
}

var productionSpoilerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bexercise\s*\d`), // "exercise 13"
	regexp.MustCompile(`(?i)\b(planted|spoiler)\b`),
}

// checkSpoilers scans comments in every .go file. Test comments may name their
// exercise and describe observable behavior, but they must not label exact
// implementations as buggy/fixed or state the repair. Production comments
// additionally may not mention exercise numbers or planted spoilers.
// Comments are parsed with the Go parser so block comments are covered and
// comment-like text inside string literals is ignored.
func checkSpoilers() []string {
	var ps []string
	for _, root := range []string{"internal", "cmd"} {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			patterns := commonSpoilerPatterns
			if !strings.HasSuffix(path, "_test.go") {
				patterns = append(patterns, productionSpoilerPatterns...)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				ps = append(ps, fmt.Sprintf("%s: cannot parse comments: %v", path, err))
				return nil
			}
			for _, group := range file.Comments {
				for _, comment := range group.List {
					for _, re := range patterns {
						if re.MatchString(comment.Text) {
							pos := fset.Position(comment.Slash)
							ps = append(ps, fmt.Sprintf("%s:%d: comment matches spoiler pattern %s: %q",
								path, pos.Line, re.String(), strings.TrimSpace(comment.Text)))
							break
						}
					}
				}
			}
			return nil
		})
	}
	return ps
}

func checkMakefile(exercises []exercise) []string {
	var ps []string
	data, err := os.ReadFile("Makefile")
	if err != nil {
		return []string{fmt.Sprintf("Makefile unreadable: %v", err)}
	}
	get := func(name string) []string {
		re := regexp.MustCompile(`(?m)^` + name + `\s*:=\s*(.+)$`)
		m := re.FindStringSubmatch(string(data))
		if m == nil {
			return nil
		}
		return strings.Fields(m[1])
	}
	wantAll := map[string]bool{}
	wantPre := map[string]bool{}
	for _, e := range exercises {
		wantAll[e.Number] = true
		if e.SolvedInMain != "" {
			wantPre[e.Number] = true
		}
	}
	if diff := setDiff(wantAll, get("EXERCISES")); diff != "" {
		ps = append(ps, "Makefile EXERCISES out of sync with the registry: "+diff)
	}
	if diff := setDiff(wantPre, get("PRESOLVED")); diff != "" {
		ps = append(ps, "Makefile PRESOLVED out of sync with solved_in_main: "+diff)
	}
	return ps
}

func checkFormatting() []string {
	var ps []string
	for _, root := range []string{"cmd", "internal", "tools"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			formatted, err := format.Source(data)
			if err != nil {
				ps = append(ps, fmt.Sprintf("%s cannot be formatted: %v", path, err))
				return nil
			}
			if !bytes.Equal(data, formatted) {
				ps = append(ps, path+" is not gofmt-formatted")
			}
			return nil
		})
		if err != nil {
			ps = append(ps, fmt.Sprintf("walking %s: %v", root, err))
		}
	}
	return ps
}

func setDiff(want map[string]bool, got []string) string {
	gotSet := map[string]bool{}
	for _, g := range got {
		gotSet[g] = true
	}
	var missing, extra []string
	for w := range want {
		if !gotSet[w] {
			missing = append(missing, w)
		}
	}
	for g := range gotSet {
		if !want[g] {
			extra = append(extra, g)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	var parts []string
	if len(missing) > 0 {
		parts = append(parts, "missing "+strings.Join(missing, ","))
	}
	if len(extra) > 0 {
		parts = append(parts, "extra "+strings.Join(extra, ","))
	}
	return strings.Join(parts, "; ")
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func pathWithin(path, directory string) bool {
	path = filepath.Clean(path)
	directory = filepath.Clean(directory)
	return path == directory || strings.HasPrefix(path, directory+string(filepath.Separator))
}

func readDir(p string) []fs.DirEntry {
	ents, err := os.ReadDir(p)
	if err != nil {
		return nil
	}
	return ents
}

func fileLines(p string) ([]string, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func numberSet(exercises []exercise) map[string]bool {
	s := map[string]bool{}
	for _, e := range exercises {
		s[e.Number] = true
	}
	return s
}
