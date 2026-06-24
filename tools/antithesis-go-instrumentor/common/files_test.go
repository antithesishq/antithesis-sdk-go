package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathFromBaseDirectory(t *testing.T) {
	// Use a temp dir so CanonicalizeDirectory's filepath.EvalSymlinks
	// has real paths to walk. Build a small tree:
	//   <tmp>/repo/
	//   <tmp>/repo/server/
	//   <tmp>/repo/server/test/
	//   <tmp>/repo/tools/inner/leaf/
	//   <tmp>/other/   (sibling, not under repo)
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	for _, d := range []string{
		filepath.Join(repo, "server", "test"),
		filepath.Join(repo, "tools", "inner", "leaf"),
		filepath.Join(tmp, "other"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		base    string
		some    string
		wantRel string
	}{
		{
			name:    "same_dir_returns_empty",
			base:    repo,
			some:    repo,
			wantRel: "",
		},
		{
			name:    "direct_child",
			base:    repo,
			some:    filepath.Join(repo, "server"),
			wantRel: "server",
		},
		{
			name:    "two_level_child",
			base:    repo,
			some:    filepath.Join(repo, "server", "test"),
			wantRel: filepath.Join("server", "test"),
		},
		{
			name:    "three_level_child",
			base:    repo,
			some:    filepath.Join(repo, "tools", "inner", "leaf"),
			wantRel: filepath.Join("tools", "inner", "leaf"),
		},
		{
			name:    "not_a_child_returns_empty",
			base:    repo,
			some:    filepath.Join(tmp, "other"),
			wantRel: "",
		},
		{
			name:    "trailing_slash_on_base_is_normalized",
			base:    repo + string(filepath.Separator),
			some:    filepath.Join(repo, "server", "test"),
			wantRel: filepath.Join("server", "test"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PathFromBaseDirectory(tc.base, tc.some)
			if got != tc.wantRel {
				t.Errorf("PathFromBaseDirectory(%q, %q) = %q, want %q", tc.base, tc.some, got, tc.wantRel)
			}
		})
	}
}

// Regression test for ENG-3940: the previous implementation used
// filepath.Match with a `baseDir/*` pattern, which does not cross path
// separators. Multi-level submodules silently returned the *absolute* path
// of someDir as the "offset", which the caller then joined onto the
// customer output dir producing customer/<input-abspath>/... directories.
func TestPathFromBaseDirectory_DeepSubmodule_NoAbsolutePath(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	deep := filepath.Join(repo, "tools", "rw-heatmaps")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	got := PathFromBaseDirectory(repo, deep)
	want := filepath.Join("tools", "rw-heatmaps")
	if got != want {
		t.Fatalf("PathFromBaseDirectory(%q, %q) = %q, want %q (regression: returning the absolute path instead of a relative offset)",
			repo, deep, got, want)
	}
	if filepath.IsAbs(got) {
		t.Fatalf("PathFromBaseDirectory returned an absolute path %q for a child under base (this is the ENG-3940 bug)", got)
	}
}

func TestPathFromBaseDirectory_Symlink(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	real := filepath.Join(repo, "real", "deep")
	if err := os.MkdirAll(real, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(repo, "alias")
	if err := os.Symlink(filepath.Join(repo, "real"), link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	// Passing the symlinked path should resolve to the real path and
	// produce a relative offset against the base.
	got := PathFromBaseDirectory(repo, filepath.Join(link, "deep"))
	want := filepath.Join("real", "deep")
	if got != want {
		t.Errorf("PathFromBaseDirectory(%q, %q) = %q, want %q", repo, filepath.Join(link, "deep"), got, want)
	}
}
