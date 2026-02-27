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
func (c *apiClient) hasSameUndoneCopyTask(ctx context.Context, srcFile, dstDir string) (bool, error) {
	tasks, err := c.listUndoneCopyTasks(ctx)
	if err != nil {
		return false, err
	}

	wantKey := buildTaskKey(srcFile, dstDir)
	for _, t := range tasks {
		key, ok := parseCopyTaskKey(t.Name)
		if !ok {
			continue
		}
		if key == wantKey {
			return true, nil
		}
	}
	return false, nil
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
