package main

import "testing"

func TestCompareSnapshotsFindsUsefulDifferences(t *testing.T) {
	a := Snapshot{
		SchemaVersion: 1,
		System:        SystemInfo{OS: "linux", Arch: "amd64", Timezone: "UTC"},
		Git:           &GitInfo{Commit: "abc", Branch: "main"},
		Tools:         map[string]Tool{"node": {Path: "/usr/bin/node", Version: "v22.1.0"}},
		Env:           EnvironmentInfo{Present: []string{"DATABASE_URL", "PATH"}, Safe: map[string]string{"TZ": "UTC"}},
	}
	b := Snapshot{
		SchemaVersion: 1,
		System:        SystemInfo{OS: "linux", Arch: "arm64", Timezone: "Asia/Tokyo"},
		Git:           &GitInfo{Commit: "def", Branch: "main"},
		Tools:         map[string]Tool{"node": {Path: "/opt/node", Version: "v20.0.0"}},
		Env:           EnvironmentInfo{Present: []string{"PATH"}, Safe: map[string]string{"TZ": "Asia/Tokyo"}},
	}
	diffs := compareSnapshots(a, b)
	keys := map[string]bool{}
	for _, d := range diffs {
		keys[d.Key] = true
	}
	for _, want := range []string{"system.arch", "git.commit", "tool.node.version", "env.DATABASE_URL.present"} {
		if !keys[want] {
			t.Fatalf("missing difference %s", want)
		}
	}
}

func TestNoDifferences(t *testing.T) {
	s := Snapshot{SchemaVersion: 1, System: SystemInfo{OS: "linux", Arch: "amd64"}, Tools: map[string]Tool{}, Env: EnvironmentInfo{Safe: map[string]string{}}}
	if got := compareSnapshots(s, s); len(got) != 0 {
		t.Fatalf("expected no differences, got %d", len(got))
	}
}
