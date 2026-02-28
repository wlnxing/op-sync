package openlistsync

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
)

type treeSnapshot struct {
	Files map[string]int64
	Dirs  map[string]struct{}
}

type copyPlanItem struct {
	RelPath string
	SrcSize int64
	DstSize int64
	Reason  string
}

// Run 执行一次目录增量同步。
func Run(ctx context.Context, cfg Config) error {
	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return err
	}
	filter, err := newPathFilter(cfg.Blacklist)
	if err != nil {
		return err
	}
	c := newAPIClient(cfg)
	if filter.count() > 0 {
		cfg.Logger.Infof("blacklist enabled with %d pattern(s)", filter.count())
	}
	if cfg.MinSizeDiff > 0 {
		cfg.Logger.Infof("min size diff enabled: %d bytes", cfg.MinSizeDiff)
	}

	cfg.Logger.Infof("scan source: %s", cfg.SrcDir)
	srcSnap, err := scanTree(ctx, c, cfg.SrcDir, filter, cfg.Logger)
	if err != nil {
		cfg.Logger.Errorf("scan source failed: %v", err)
		return fmt.Errorf("scan source failed: %w", err)
	}

	cfg.Logger.Infof("scan target: %s", cfg.DstDir)
	dstSnap, err := scanTree(ctx, c, cfg.DstDir, filter, cfg.Logger)
	if err != nil {
		if isNotFoundErr(err) {
			cfg.Logger.Infof("target dir not found, create: %s", cfg.DstDir)
			if err := c.mkdir(ctx, cfg.DstDir); err != nil {
				cfg.Logger.Errorf("create target dir failed: %v", err)
				return fmt.Errorf("create target dir failed: %w", err)
			}
			dstSnap = &treeSnapshot{
				Files: map[string]int64{},
				Dirs:  map[string]struct{}{"": {}},
			}
		} else {
			cfg.Logger.Errorf("scan target failed: %v", err)
			return fmt.Errorf("scan target failed: %w", err)
		}
	}

	plan, unchanged := buildPlan(srcSnap.Files, dstSnap.Files, cfg.MinSizeDiff)
	cfg.Logger.Infof("source files: %d, target files: %d", len(srcSnap.Files), len(dstSnap.Files))
	cfg.Logger.Infof("to copy: %d, unchanged/skipped: %d", len(plan), unchanged)

	if len(plan) == 0 {
		cfg.Logger.Infof("nothing to sync")
		return nil
	}
	for _, item := range plan {
		cfg.Logger.Debugf("PLAN %s | src=%d dst=%d | %s", item.RelPath, item.SrcSize, item.DstSize, item.Reason)
	}
	if cfg.DryRun {
		cfg.Logger.Infof("dry-run enabled, no copy submitted")
		return nil
	}

	knownDstDirs := make(map[string]struct{}, len(dstSnap.Dirs)+1)
	for relDir := range dstSnap.Dirs {
		knownDstDirs[joinRootWithRel(cfg.DstDir, relDir)] = struct{}{}
	}
	knownDstDirs[cfg.DstDir] = struct{}{}

	var submitted, skippedDup, failed int

	for _, item := range plan {
		srcFile := joinRootWithRel(cfg.SrcDir, item.RelPath)
		dstFile := joinRootWithRel(cfg.DstDir, item.RelPath)
		dstParent := normalizeOLPath(path.Dir(dstFile))

		if err := ensureDir(ctx, c, dstParent, knownDstDirs); err != nil {
			failed++
			cfg.Logger.Errorf("mkdir failed %s: %v", dstParent, err)
			continue
		}

		hasSameTask, err := c.hasSameUndoneCopyTask(ctx, srcFile, dstParent)
		if err != nil {
			failed++
			cfg.Logger.Errorf("check undone task failed %s -> %s: %v", srcFile, dstParent, err)
			continue
		}
		if hasSameTask {
			skippedDup++
			cfg.Logger.Infof("skip duplicate task %s -> %s", srcFile, dstParent)
			continue
		}

		srcParent := normalizeOLPath(path.Dir(srcFile))
		name := path.Base(srcFile)
		if err := c.copyFile(ctx, srcParent, dstParent, name, true); err != nil {
			failed++
			cfg.Logger.Errorf("copy failed %s -> %s: %v", srcFile, dstParent, err)
			continue
		}
		submitted++
		cfg.Logger.Infof("copy %s -> %s (%s)", srcFile, dstParent, item.Reason)
	}

	cfg.Logger.Infof("done: submitted=%d skipped_duplicate_task=%d failed=%d", submitted, skippedDup, failed)
	if failed > 0 {
		cfg.Logger.Errorf("sync finished with %d failed items", failed)
		return fmt.Errorf("sync finished with %d failed items", failed)
	}
	return nil
}

// scanTree 通过 OpenList 的 list API 递归遍历目录，构建：
// 1) 以相对路径为 key 的文件大小索引
// 2) 以相对路径为 key 的目录集合
func scanTree(ctx context.Context, c *apiClient, root string, filter *pathFilter, logger *Logger) (*treeSnapshot, error) {
	snap := &treeSnapshot{
		Files: make(map[string]int64),
		Dirs:  map[string]struct{}{"": {}},
	}
	queue := []string{""}

	for len(queue) > 0 {
		relDir := queue[0]
		queue = queue[1:]
		absDir := joinRootWithRel(root, relDir)
		logger.Debugf("scanning directory: %s", absDir)

		entries, err := c.listAllEntries(ctx, absDir)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", absDir, err)
		}

		for _, obj := range entries {
			relPath := obj.Name
			if relDir != "" {
				relPath = path.Join(relDir, obj.Name)
			}
			if filter.match(relPath) {
				logger.Debugf("skip by blacklist: %s", relPath)
				continue
			}
			if obj.IsDir {
				snap.Dirs[relPath] = struct{}{}
				queue = append(queue, relPath)
				continue
			}
			snap.Files[relPath] = obj.Size
		}
	}

	return snap, nil
}

// buildPlan 对比源/目标文件索引并生成复制计划。
// 规则：
// - 目标不存在：复制
// - 同路径且源文件更大：覆盖复制
// - 其他情况：跳过
func buildPlan(srcFiles, dstFiles map[string]int64, minSizeDiff int64) ([]copyPlanItem, int) {
	plan := make([]copyPlanItem, 0)
	unchanged := 0

	for rel, srcSize := range srcFiles {
		dstSize, ok := dstFiles[rel]
		if !ok {
			plan = append(plan, copyPlanItem{
				RelPath: rel,
				SrcSize: srcSize,
				DstSize: -1,
				Reason:  "target missing",
			})
			continue
		}
		diff := srcSize - dstSize
		if diff > 0 && diff >= minSizeDiff {
			plan = append(plan, copyPlanItem{
				RelPath: rel,
				SrcSize: srcSize,
				DstSize: dstSize,
				Reason:  fmt.Sprintf("source larger by %d bytes, overwrite", diff),
			})
			continue
		}
		unchanged++
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].RelPath < plan[j].RelPath
	})
	return plan, unchanged
}

// ensureDir 在目录未知时递归创建目录。
// known 用于避免共享父目录被重复 mkdir。
func ensureDir(ctx context.Context, c *apiClient, absDir string, known map[string]struct{}) error {
	absDir = normalizeOLPath(absDir)
	if _, ok := known[absDir]; ok {
		return nil
	}

	parent := path.Dir(absDir)
	if parent != absDir {
		if err := ensureDir(ctx, c, parent, known); err != nil {
			return err
		}
	}

	if err := c.mkdir(ctx, absDir); err != nil {
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "exist") {
			return err
		}
	}
	known[absDir] = struct{}{}
	return nil
}
