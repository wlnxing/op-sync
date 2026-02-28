package openlistsync

import (
	"context"
	"path"
	"regexp"
	"strings"
)

var copyTaskNameRe = regexp.MustCompile(`^copy \[(.+)\]\((.+)\) to \[(.+)\]\((.+)\)$`)

// hasSameUndoneCopyTask 检查 OpenList 未完成复制任务中是否已存在等价任务，
// 用于避免重复提交。
func (c *apiClient) hasSameUndoneCopyTask(ctx context.Context, srcFile, dstDir, userBasePath string) (bool, error) {
	tasks, err := c.listUndoneCopyTasks(ctx)
	if err != nil {
		return false, err
	}

	wantKeys := buildWantTaskKeys(srcFile, dstDir, userBasePath)
	for _, t := range tasks {
		key, ok := parseCopyTaskKey(t.Name)
		if !ok {
			continue
		}
		if _, ok := wantKeys[key]; ok {
			return true, nil
		}
	}
	return false, nil
}

// buildWantTaskKeys 构造待匹配任务 key：
// 1) 用户视角路径（配置里的 src/dst）
// 2) root 视角路径（拼上当前用户 base_path）
func buildWantTaskKeys(srcFile, dstDir, userBasePath string) map[string]struct{} {
	keys := map[string]struct{}{
		buildTaskKey(srcFile, dstDir): {},
	}

	base := normalizeOLPath(userBasePath)
	if base == "/" {
		return keys
	}
	srcWithBase := applyBasePath(base, srcFile)
	dstWithBase := applyBasePath(base, dstDir)
	keys[buildTaskKey(srcWithBase, dstWithBase)] = struct{}{}
	return keys
}

// parseCopyTaskKey 解析 OpenList 复制任务名：
// copy [srcMount](srcActualPath) to [dstMount](dstActualPath)
// 并归一化为 "srcFile->dstDir"。
func parseCopyTaskKey(taskName string) (string, bool) {
	m := copyTaskNameRe.FindStringSubmatch(strings.TrimSpace(taskName))
	if len(m) != 5 {
		return "", false
	}

	src := joinMountAndActual(m[1], m[2])
	dst := joinMountAndActual(m[3], m[4])
	return buildTaskKey(src, dst), true
}

func buildTaskKey(srcFile, dstDir string) string {
	return normalizeOLPath(srcFile) + "->" + normalizeOLPath(dstDir)
}

func joinMountAndActual(mountPath, actualPath string) string {
	mountPath = normalizeOLPath(mountPath)
	actualPath = strings.TrimSpace(actualPath)
	if actualPath == "" {
		actualPath = "/"
	}
	if !strings.HasPrefix(actualPath, "/") {
		actualPath = "/" + actualPath
	}
	actualPath = path.Clean(actualPath)

	if mountPath == "/" {
		return actualPath
	}
	if actualPath == "/" {
		return mountPath
	}
	return normalizeOLPath(strings.TrimSuffix(mountPath, "/") + "/" + strings.TrimPrefix(actualPath, "/"))
}

func applyBasePath(basePath, p string) string {
	basePath = normalizeOLPath(basePath)
	p = normalizeOLPath(p)
	if basePath == "/" {
		return p
	}

	if p == basePath || strings.HasPrefix(p, basePath+"/") {
		return p
	}
	return normalizeOLPath(path.Join(basePath, strings.TrimPrefix(p, "/")))
}
