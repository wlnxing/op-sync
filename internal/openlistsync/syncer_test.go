package openlistsync

import "testing"

func TestParseCopyTaskKey(t *testing.T) {
	tests := []struct {
		name    string
		task    string
		wantKey string
		wantOK  bool
	}{
		{
			name:    "relative actual path",
			task:    "copy [/src](dir/a.txt) to [/dst](out)",
			wantKey: "/src/dir/a.txt->/dst/out",
			wantOK:  true,
		},
		{
			name:    "absolute actual path",
			task:    "copy [/src](/dir/a.txt) to [/dst](/out)",
			wantKey: "/src/dir/a.txt->/dst/out",
			wantOK:  true,
		},
		{
			name:    "invalid format",
			task:    "copy something else",
			wantKey: "",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCopyTaskKey(tt.task)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.wantKey {
				t.Fatalf("key = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestBuildPlan(t *testing.T) {
	src := map[string]int64{
		"a.txt":     10,
		"b.txt":     5,
		"sub/c.txt": 8,
	}
	dst := map[string]int64{
		"a.txt": 3,
		"b.txt": 6,
	}

	plan, unchanged := buildPlan(src, dst, 0)
	if unchanged != 1 {
		t.Fatalf("unchanged = %d, want 1", unchanged)
	}
	if len(plan) != 2 {
		t.Fatalf("plan length = %d, want 2", len(plan))
	}
	if plan[0].RelPath != "a.txt" || plan[0].Reason != "source larger by 7 bytes, overwrite" {
		t.Fatalf("plan[0] = %+v, unexpected", plan[0])
	}
	if plan[1].RelPath != "sub/c.txt" || plan[1].Reason != "target missing" {
		t.Fatalf("plan[1] = %+v, unexpected", plan[1])
	}
}

func TestBuildPlanWithMinSizeDiff(t *testing.T) {
	src := map[string]int64{
		"a.txt": 10,
	}
	dst := map[string]int64{
		"a.txt": 7,
	}

	plan, unchanged := buildPlan(src, dst, 4)
	if unchanged != 1 {
		t.Fatalf("unchanged = %d, want 1", unchanged)
	}
	if len(plan) != 0 {
		t.Fatalf("plan length = %d, want 0", len(plan))
	}

	plan, unchanged = buildPlan(src, dst, 3)
	if unchanged != 0 {
		t.Fatalf("unchanged = %d, want 0", unchanged)
	}
	if len(plan) != 1 {
		t.Fatalf("plan length = %d, want 1", len(plan))
	}
}

func TestPathFilterMatch(t *testing.T) {
	f, err := newPathFilter([]string{"*.tmp", "cache/*", "sub/ignore.txt", "node_modules"})
	if err != nil {
		t.Fatalf("newPathFilter error: %v", err)
	}

	tests := []struct {
		relPath string
		want    bool
	}{
		{relPath: "a.tmp", want: true},
		{relPath: "cache/a.txt", want: true},
		{relPath: "sub/ignore.txt", want: true},
		{relPath: "dir/node_modules", want: true},
		{relPath: "keep.txt", want: false},
		{relPath: "cache/sub/a.txt", want: false},
	}

	for _, tt := range tests {
		got := f.match(tt.relPath)
		if got != tt.want {
			t.Fatalf("match(%q)=%v, want=%v", tt.relPath, got, tt.want)
		}
	}
}

func TestPathFilterInvalidPattern(t *testing.T) {
	_, err := newPathFilter([]string{"["})
	if err == nil {
		t.Fatalf("expected invalid pattern error")
	}
}
