package openlistsync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type apiClient struct {
	baseURL    string
	token      string
	perPage    int
	logger     *Logger
	httpClient *http.Client
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type fsObj struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"is_dir"`
}

type fsListData struct {
	Content []fsObj `json:"content"`
	Total   int64   `json:"total"`
}

type taskInfo struct {
	Name string `json:"name"`
}

type copyReq struct {
	SrcDir       string   `json:"src_dir"`
	DstDir       string   `json:"dst_dir"`
	Names        []string `json:"names"`
	Overwrite    bool     `json:"overwrite"`
	SkipExisting bool     `json:"skip_existing"`
	Merge        bool     `json:"merge"`
}

type mkdirReq struct {
	Path string `json:"path"`
}

func newAPIClient(cfg Config) *apiClient {
	return &apiClient{
		baseURL: cfg.BaseURL,
		token:   cfg.Token,
		perPage: cfg.PerPage,
		logger:  cfg.Logger,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *apiClient) listUndoneCopyTasks(ctx context.Context) ([]taskInfo, error) {
	var tasks []taskInfo
	if err := c.requestJSON(ctx, http.MethodGet, "/api/admin/task/copy/undone", nil, &tasks); err != nil {
		return nil, err
	}
	if tasks == nil {
		return []taskInfo{}, nil
	}
	return tasks, nil
}

func (c *apiClient) copyFile(ctx context.Context, srcDir, dstDir, name string, overwrite bool) error {
	req := copyReq{
		SrcDir:       normalizeOLPath(srcDir),
		DstDir:       normalizeOLPath(dstDir),
		Names:        []string{name},
		Overwrite:    overwrite,
		SkipExisting: false,
		Merge:        false,
	}
	return c.requestJSON(ctx, http.MethodPost, "/api/fs/copy", req, nil)
}

func (c *apiClient) mkdir(ctx context.Context, p string) error {
	return c.requestJSON(ctx, http.MethodPost, "/api/fs/mkdir", mkdirReq{Path: normalizeOLPath(p)}, nil)
}

func (c *apiClient) listAllEntries(ctx context.Context, p string) ([]fsObj, error) {
	p = normalizeOLPath(p)
	var all []fsObj
	page := 1

	for {
		req := map[string]any{
			"path":     p,
			"page":     page,
			"per_page": c.perPage,
		}
		var data fsListData
		if err := c.requestJSON(ctx, http.MethodPost, "/api/fs/list", req, &data); err != nil {
			return nil, err
		}
		if data.Content == nil {
			data.Content = []fsObj{}
		}
		all = append(all, data.Content...)

		if len(data.Content) == 0 || int64(len(all)) >= data.Total {
			break
		}
		page++
	}

	return all, nil
}

// requestJSON 发送 OpenList API 请求，并解包标准响应：
// {"code":..., "message":..., "data":...}
// code 非 200 一律按错误处理。
func (c *apiClient) requestJSON(ctx context.Context, method, apiPath string, payload any, out any) error {
	c.logger.Debugf("request %s %s", method, apiPath)

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+apiPath, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	var envelope apiResp
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("decode response failed, status=%d body=%q", resp.StatusCode, truncateBytes(respBody, 300))
	}

	if envelope.Code != 200 {
		c.logger.Errorf("api %s failed: code=%d message=%s", apiPath, envelope.Code, envelope.Message)
		return fmt.Errorf("api %s failed: code=%d message=%s", apiPath, envelope.Code, envelope.Message)
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode response data for %s: %w", apiPath, err)
	}
	return nil
}
