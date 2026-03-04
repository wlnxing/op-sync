package openlistsync

import "testing"

func TestNormalizeConfigOutputDefaultToDst(t *testing.T) {
	cfg, err := normalizeConfig(Config{
		BaseURL: "http://localhost:35244",
		Token:   "token",
		SrcDir:  "/src",
		DstDir:  "/dst",
	})
	if err != nil {
		t.Fatalf("normalizeConfig error: %v", err)
	}
	if cfg.OutputDir != "/dst" {
		t.Fatalf("output_dir=%q, want /dst", cfg.OutputDir)
	}
}

func TestNormalizeConfigOutputNormalize(t *testing.T) {
	cfg, err := normalizeConfig(Config{
		BaseURL:   "http://localhost:35244",
		Token:     "token",
		SrcDir:    "/src",
		DstDir:    "/dst",
		OutputDir: "out/sub",
	})
	if err != nil {
		t.Fatalf("normalizeConfig error: %v", err)
	}
	if cfg.OutputDir != "/out/sub" {
		t.Fatalf("output_dir=%q, want /out/sub", cfg.OutputDir)
	}
}
