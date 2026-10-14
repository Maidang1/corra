package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HostingConfig configures the GitHub image host (图床).
type HostingConfig struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"-"` // stored in a 0600 sidecar file, never here
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Branch  string `json:"branch"`
	Prefix  string `json:"prefix"`  // path prefix, e.g. images/2026/10
	CDN     string `json:"cdn"`     // e.g. https://cdn.jsdelivr.net/gh/{owner}/{repo}@{branch}
	Message string `json:"message"` // commit message template
}

// DefaultCDN is the jsDelivr template used when none is configured.
const DefaultCDN = "https://cdn.jsdelivr.net/gh/{owner}/{repo}@{branch}"

// hostingStore persists the hosting configuration.
type hostingStore struct {
	dir string
}

func newHostingStore(dir string) *hostingStore { return &hostingStore{dir: dir} }

func (s *hostingStore) load(cfg *HostingConfig) error {
	data, err := os.ReadFile(filepath.Join(s.dir, "hosting.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	tok, err := os.ReadFile(filepath.Join(s.dir, "hosting.token"))
	if err == nil {
		cfg.Token = strings.TrimSpace(string(tok))
	}
	return nil
}

func (s *hostingStore) save(cfg HostingConfig) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	tok := cfg.Token
	cfg.Token = ""
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.dir, "hosting.json"), data, 0o600); err != nil {
		return err
	}
	if tok != "" {
		if err := os.WriteFile(filepath.Join(s.dir, "hosting.token"), []byte(tok), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// LoadHosting reads the hosting configuration from the app config dir.
func LoadHosting(dir string) HostingConfig {
	cfg := HostingConfig{Enabled: false, Branch: "main", CDN: DefaultCDN, Message: "chore: add image"}
	_ = newHostingStore(dir).load(&cfg)
	return cfg
}

// SaveHosting persists the hosting configuration.
func SaveHosting(dir string, cfg HostingConfig) error {
	return newHostingStore(dir).save(cfg)
}

// UploadResult is what the GitHub uploader returns.
type UploadResult struct {
	URL string // the CDN URL to insert into the markdown
	Raw string // the direct github raw URL (fallback)
}

// UploadImageToGitHub uploads image bytes to the configured GitHub repo
// through the Contents API and returns the CDN URL.
func UploadImageToGitHub(cfg HostingConfig, name string, data []byte) (*UploadResult, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("图床未配置：请先在「文件 → 图床设置」中填写 GitHub token")
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		return nil, fmt.Errorf("图床未配置：请填写目标仓库 owner/repo")
	}
	branch := cfg.Branch
	if branch == "" {
		branch = "main"
	}
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix == "" {
		prefix = time.Now().Format("images/2006/01")
	}
	path := fmt.Sprintf("%s/%s-%s", prefix, time.Now().Format("20060102150405"), name)

	body := map[string]any{
		"message": renderCommitMessage(cfg.Message, path),
		"content": base64.StdEncoding.EncodeToString(data),
		"branch":  branch,
	}
	payload, _ := json.Marshal(body)
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		cfg.Owner, cfg.Repo, path)
	req, err := http.NewRequest(http.MethodPut, api, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("上传失败（网络错误）：%w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("GitHub 上传失败（%s）：%.120s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	raw := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", cfg.Owner, cfg.Repo, branch, path)
	cdn := cfg.CDN
	if cdn == "" {
		cdn = DefaultCDN
	}
	url := strings.NewReplacer(
		"{owner}", cfg.Owner,
		"{repo}", cfg.Repo,
		"{branch}", branch,
		"{path}", path,
	).Replace(cdn)
	if !strings.Contains(cdn, "{path}") {
		url = strings.TrimRight(cdn, "/") + "/" + path
	}
	return &UploadResult{URL: url, Raw: raw}, nil
}

func renderCommitMessage(tmpl, path string) string {
	if tmpl == "" {
		tmpl = "chore: add image"
	}
	return strings.NewReplacer("{path}", path, "{name}", filepath.Base(path)).Replace(tmpl)
}
