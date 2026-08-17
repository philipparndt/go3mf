package buildplan

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig writes a minimal valid YAML config naming the given output and
// returns its path.
func writeConfig(t *testing.T, output string) string {
	t.Helper()
	dir := t.TempDir()

	scad := filepath.Join(dir, "a.scad")
	if err := os.WriteFile(scad, []byte("cube([1,1,1]);\n"), 0o644); err != nil {
		t.Fatalf("writing scad: %v", err)
	}

	path := filepath.Join(dir, "recipe.yaml")
	body := "output: " + output + "\nobjects:\n  - name: thing\n    parts:\n      - name: platform\n        file: a.scad\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}

// loadStepOf returns the plan's LoadYAMLStep, which is the step that decides
// what a YAML build writes to.
func loadStepOf(t *testing.T, plan *BuildPlan) *LoadYAMLStep {
	t.Helper()
	for _, step := range plan.Steps {
		if load, ok := step.(*LoadYAMLStep); ok {
			return load
		}
	}
	t.Fatal("plan has no LoadYAMLStep")
	return nil
}

// TestYAMLPlanTakesTheRequestedOutput covers the bug where `go3mf build
// recipe.yaml -o somewhere.3mf` wrote the name in the recipe and ignored -o.
// Every other kind of input took its name from the flag; the YAML branch was
// the only one that dropped it, and it dropped it silently.
func TestYAMLPlanTakesTheRequestedOutput(t *testing.T) {
	config := writeConfig(t, "from_the_recipe.3mf")

	plan, err := NewPlanner().CreatePlan([]string{config}, nil, "wanted_here.3mf")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if plan.OutputFile != "wanted_here.3mf" {
		t.Errorf("plan output = %q, want wanted_here.3mf", plan.OutputFile)
	}
	if got := loadStepOf(t, plan).RequestedOutput; got != "wanted_here.3mf" {
		t.Errorf("step's requested output = %q, want wanted_here.3mf", got)
	}
}

// TestYAMLPlanWithoutRequestedOutputLeavesItToTheConfig is the other half, and
// the reason -o is not simply defaulted before the branch: a config that names
// its own output must keep it when nobody passed the flag. Defaulting early is
// what made these two cases indistinguishable.
func TestYAMLPlanWithoutRequestedOutputLeavesItToTheConfig(t *testing.T) {
	config := writeConfig(t, "from_the_recipe.3mf")

	plan, err := NewPlanner().CreatePlan([]string{config}, nil, "")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if plan.OutputFile != "" {
		t.Errorf("plan output = %q, want it left empty for the config to fill", plan.OutputFile)
	}
	if got := loadStepOf(t, plan).RequestedOutput; got != "" {
		t.Errorf("step's requested output = %q, want empty", got)
	}
}

// TestNonYAMLPlanStillDefaultsItsOutput guards the behaviour the early default
// was there for: an input with no opinion of its own still gets a name.
func TestNonYAMLPlanStillDefaultsItsOutput(t *testing.T) {
	dir := t.TempDir()
	scad := filepath.Join(dir, "a.scad")
	if err := os.WriteFile(scad, []byte("cube([1,1,1]);\n"), 0o644); err != nil {
		t.Fatalf("writing scad: %v", err)
	}

	plan, err := NewPlanner().CreatePlan([]string{scad}, nil, "")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if plan.OutputFile != DefaultOutputFile {
		t.Errorf("plan output = %q, want %q", plan.OutputFile, DefaultOutputFile)
	}
}
