package main

import (
	"debug/buildinfo"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStaticARMv6Binary(t *testing.T) {
	// Arrange

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	bin := filepath.Join(t.TempDir(), "myapp")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Dir(file)
	cmd.Env = replaceEnv(os.Environ(), map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        "linux",
		"GOARCH":      "arm",
		"GOARM":       "6",
	})

	// Act

	out, err := cmd.CombinedOutput()

	// Assert

	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	info, err := buildinfo.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, setting := range info.Settings {
		got[setting.Key] = setting.Value
	}
	want := map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        "linux",
		"GOARCH":      "arm",
		"GOARM":       "6",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s=%q, want %q", key, got[key], value)
		}
	}

	f, err := elf.Open(bin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Machine != elf.EM_ARM {
		t.Fatalf("machine=%v, want EM_ARM", f.Machine)
	}
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_INTERP {
			t.Fatal("binary has a dynamic linker")
		}
	}
}

func replaceEnv(base []string, overrides map[string]string) []string {
	skip := map[string]bool{}
	for key := range overrides {
		skip[key] = true
	}
	out := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok && skip[key] {
			continue
		}
		out = append(out, entry)
	}
	for key, value := range overrides {
		out = append(out, key+"="+value)
	}
	return out
}
