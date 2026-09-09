package downloader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const maxDownloadRetries = 5

type partialMetadata struct {
	URL          string `json:"url"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	Total        int64  `json:"total,omitempty"`
}

type Events struct {
	Status          string
	Progress        string
	Done            string
	Error           string
	ProgressFactory func(DownloadProgress) any
	StatusFactory   func(status string, dest string) any
	DoneFactory     func(dest string) any
	ErrorFactory    func(err string, dest string) any
}

type Options struct {
	Throttle       time.Duration
	Resume         bool
	RemoveOnCancel bool
}

type DownloadProgress struct {
	Downloaded int64
	Total      int64
	Dest       string
}

type Manager struct {
	mu     sync.Mutex
	tasks  map[string]*state
	events Events
	opts   Options
}

type state struct {
	ctx         context.Context
	url         string
	dest        string
	expectedMD5 string
	retryCount  int
	total       int64
	downloaded  int64
	paused      bool
	cancelled   bool
	running     bool
	cancelFn    context.CancelFunc
}

func NewManager(events Events, opts Options) *Manager {
	if opts.Throttle <= 0 {
		opts.Throttle = 250 * time.Millisecond
	}
	return &Manager{events: events, opts: opts, tasks: map[string]*state{}}
}

func (m *Manager) Start(ctx context.Context, src string, dest string, md5sum string) string {
	dir := filepath.Dir(dest)
	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	m.mu.Lock()
	if m.tasks == nil {
		m.tasks = map[string]*state{}
	}
	if existing, ok := m.tasks[dest]; ok && existing != nil && existing.running {
		m.mu.Unlock()
		m.emitStatus("started", dest)
		return dest
	}
	local := &state{ctx: ctx, url: src, dest: dest, expectedMD5: md5sum}
	m.tasks[dest] = local
	m.mu.Unlock()
	go m.run(local)
	m.emitStatus("started", dest)
	return dest
}

func (m *Manager) Pause() {
	m.mu.Lock()
	for _, st := range m.tasks {
		if st == nil {
			continue
		}
		st.paused = true
		if st.cancelFn != nil {
			st.cancelFn()
		}
		m.emitStatus("paused", st.dest)
	}
	m.mu.Unlock()
}

func (m *Manager) Resume() {
	m.mu.Lock()
	for _, st := range m.tasks {
		if st == nil || st.cancelled {
			continue
		}
		if st.running {
			continue
		}
		st.paused = false
		go m.run(st)
		m.emitStatus("resumed", st.dest)
	}
	m.mu.Unlock()
}

func (m *Manager) Cancel() {
	m.mu.Lock()
	for dest, st := range m.tasks {
		if st == nil {
			continue
		}
		st.cancelled = true
		if st.cancelFn != nil {
			st.cancelFn()
		}
		if !st.running {
			delete(m.tasks, dest)
		}
		m.emitStatus("cancelled", st.dest)
	}
	m.mu.Unlock()
}

func (m *Manager) PauseTask(dest string) {
	m.mu.Lock()
	if st, ok := m.tasks[dest]; ok && st != nil {
		st.paused = true
		if st.cancelFn != nil {
			st.cancelFn()
		}
		m.emitStatus("paused", st.dest)
	}
	m.mu.Unlock()
}

func (m *Manager) ResumeTask(dest string) {
	m.mu.Lock()
	if st, ok := m.tasks[dest]; ok && st != nil && !st.cancelled && !st.running {
		st.paused = false
		go m.run(st)
		m.emitStatus("resumed", st.dest)
	}
	m.mu.Unlock()
}

func (m *Manager) CancelTask(dest string) {
	m.mu.Lock()
	if st, ok := m.tasks[dest]; ok && st != nil {
		st.cancelled = true
		if st.cancelFn != nil {
			st.cancelFn()
		}
		if !st.running {
			delete(m.tasks, dest)
		}
		m.emitStatus("cancelled", st.dest)
	}
	m.mu.Unlock()
}

func (m *Manager) run(s *state) {
	m.mu.Lock()
	local := m.tasks[s.dest]
	if local != s {
		m.mu.Unlock()
		return
	}
	if local == nil || local.cancelled || local.paused {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	for attempt := 0; ; attempt++ {
		var cur int64
		downloadDest := local.dest + ".download"
		metadataDest := downloadDest + ".meta"
		metadata := partialMetadata{URL: local.url}
		if m.opts.Resume {
			if fi, err := os.Stat(downloadDest); err == nil {
				cur = fi.Size()
			}
			metadata = readMetadata(metadataDest)
			if metadata.URL != "" && metadata.URL != local.url {
				cur = 0
				_ = os.Remove(downloadDest)
				_ = os.Remove(metadataDest)
				metadata = partialMetadata{URL: local.url}
			}
			if metadata.Total > 0 && cur > metadata.Total {
				cur = 0
				_ = os.Remove(downloadDest)
				_ = os.Remove(metadataDest)
			}
		}
		local.downloaded = cur
		ctx, cancel := context.WithCancel(local.ctx)
		m.mu.Lock()
		local.cancelFn = cancel
		local.running = true
		m.tasks[local.dest] = local
		m.mu.Unlock()

		req, err := http.NewRequestWithContext(ctx, "GET", local.url, nil)
		if err != nil {
			m.emitError(err.Error(), local.dest)
			m.finishRunning(local)
			return
		}
		httpx.ApplyDefaultHeaders(req)
		if m.opts.Resume && cur > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", cur))
			if metadata.ETag != "" {
				req.Header.Set("If-Range", metadata.ETag)
			} else if metadata.LastModified != "" {
				req.Header.Set("If-Range", metadata.LastModified)
			}
		}
		resp, err := httpx.Do(req)
		if err != nil {
			if !m.retry(local, attempt, err) {
				m.emitError(err.Error(), local.dest)
				m.finishRunning(local)
				return
			}
			continue
		}

		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && cur > 0 {
			resp.Body.Close()
			_ = os.Remove(downloadDest)
			_ = os.Remove(metadataDest)
			if attempt < maxDownloadRetries {
				continue
			}
			m.emitError(fmt.Sprintf("HTTP %s", resp.Status), local.dest)
			m.finishRunning(local)
			return
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			m.emitError(fmt.Sprintf("HTTP %s", resp.Status), local.dest)
			m.finishRunning(local)
			return
		}

		if m.opts.Resume && cur > 0 && resp.StatusCode == http.StatusOK {
			_ = os.Remove(downloadDest)
			cur = 0
			metadata = partialMetadata{URL: local.url}
			cur = 0
			local.downloaded = 0
		}
		metadata.URL = local.url
		metadata.ETag = resp.Header.Get("ETag")
		metadata.LastModified = resp.Header.Get("Last-Modified")

		total := resp.ContentLength
		if m.opts.Resume && total > 0 && cur > 0 {
			if cr := resp.Header.Get("Content-Range"); cr != "" {
				if idx := strings.LastIndex(cr, "/"); idx != -1 {
					if all := cr[idx+1:]; all != "*" {
						if v, e := parseInt64(all); e == nil {
							total = v
						}
					}
				}
			} else {
				total = cur + total
			}
		}
		if total <= 0 && metadata.Total > 0 {
			total = metadata.Total
		}
		metadata.Total = total
		if m.opts.Resume {
			_ = writeMetadata(metadataDest, metadata)
		}
		local.total = total

		flags := os.O_CREATE | os.O_WRONLY
		if cur == 0 {
			flags |= os.O_TRUNC
		} else {
			flags |= os.O_APPEND
		}
		f, err := os.OpenFile(downloadDest, flags, 0o644)
		if err != nil {
			resp.Body.Close()
			m.emitError(err.Error(), local.dest)
			m.finishRunning(local)
			return
		}
		{
			payload := any(DownloadProgress{Downloaded: local.downloaded, Total: local.total, Dest: local.dest})
			if m.events.ProgressFactory != nil {
				payload = m.events.ProgressFactory(DownloadProgress{Downloaded: local.downloaded, Total: local.total, Dest: local.dest})
			}
			application.Get().Event.Emit(m.events.Progress, payload)
		}
		buf := make([]byte, 128*1024)
		lastEmit := time.Now()

		var loopErr error
		for {
			if local.cancelled || local.paused {
				_ = f.Sync()
				_ = f.Close()
				resp.Body.Close()
				if local.cancelled && m.opts.RemoveOnCancel && downloadDest != "" {
					_ = os.Remove(downloadDest)
				}
				m.finishRunning(local)
				if local.cancelled {
					m.emitStatus("cancelled", local.dest)
				}
				return
			}
			n, er := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.Write(buf[:n]); werr != nil {
					loopErr = werr
					break
				}
				local.downloaded += int64(n)
				if time.Since(lastEmit) >= m.opts.Throttle {
					payload := any(DownloadProgress{Downloaded: local.downloaded, Total: local.total, Dest: local.dest})
					if m.events.ProgressFactory != nil {
						payload = m.events.ProgressFactory(DownloadProgress{Downloaded: local.downloaded, Total: local.total, Dest: local.dest})
					}
					application.Get().Event.Emit(m.events.Progress, payload)
					lastEmit = time.Now()
				}
			}
			if er != nil {
				if er == io.EOF {
					loopErr = nil
				} else {
					loopErr = er
				}
				break
			}
		}
		if loopErr != nil {
			_ = f.Sync()
		}
		_ = f.Close()
		resp.Body.Close()

		if loopErr != nil {
			if ctx.Err() == context.Canceled || local.cancelled {
				if m.opts.RemoveOnCancel && downloadDest != "" {
					_ = os.Remove(downloadDest)
				}
				m.emitStatus("cancelled", local.dest)
			} else if m.retry(local, attempt, loopErr) {
				continue
			} else {
				m.emitError(loopErr.Error(), local.dest)
			}
			m.finishRunning(local)
			return
		}

		// Download complete, verify MD5
		if local.expectedMD5 != "" {
			m.emitStatus("verifying", local.dest)
			if hash, err := calculateMD5(downloadDest); err == nil {
				if !strings.EqualFold(hash, local.expectedMD5) {
					// Mismatch
					_ = os.Remove(downloadDest)
					local.retryCount++
					if local.retryCount < 3 {
						m.emitError(fmt.Sprintf("MD5 mismatch (attempt %d/3), retrying...", local.retryCount), local.dest)
						m.emitStatus("started", local.dest)
						time.Sleep(1 * time.Second)
						continue
					} else {
						m.emitError("ERR_MD5_MISMATCH", local.dest)
						m.finishRunning(local)
						return
					}
				}
			} else {
				// Failed to calculate MD5, maybe treat as error?
				m.emitError(fmt.Sprintf("MD5 calculation failed: %v", err), local.dest)
				m.finishRunning(local)
				return
			}
		}

		if err := os.Rename(downloadDest, local.dest); err != nil {
			m.emitError(err.Error(), local.dest)
		} else {
			_ = os.Remove(metadataDest)
			payload := any(DownloadProgress{Downloaded: local.downloaded, Total: local.total, Dest: local.dest})
			if m.events.ProgressFactory != nil {
				payload = m.events.ProgressFactory(DownloadProgress{Downloaded: local.downloaded, Total: local.total, Dest: local.dest})
			}
			application.Get().Event.Emit(m.events.Progress, payload)
			m.emitDone(local.dest)
		}
		m.finishRunning(local)
		return
	}
}

func (m *Manager) retry(s *state, attempt int, err error) bool {
	if attempt >= maxDownloadRetries || s.cancelled || s.paused {
		return false
	}
	m.emitStatus("retrying", s.dest)
	backoff := time.Duration(1<<attempt) * 500 * time.Millisecond
	jitter := time.Duration(time.Now().UnixNano()%250) * time.Millisecond
	timer := time.NewTimer(backoff + jitter)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-s.ctx.Done():
		return false
	}
}

func readMetadata(path string) partialMetadata {
	data, err := os.ReadFile(path)
	if err != nil {
		return partialMetadata{}
	}
	var metadata partialMetadata
	if json.Unmarshal(data, &metadata) != nil {
		return partialMetadata{}
	}
	return metadata
}

func writeMetadata(path string, metadata partialMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func calculateMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (m *Manager) finishRunning(s *state) {
	m.mu.Lock()
	s.running = false
	if s.cancelled || !s.paused {
		if m.tasks != nil {
			if cur, ok := m.tasks[s.dest]; ok && cur == s {
				delete(m.tasks, s.dest)
			}
		}
	}
	m.mu.Unlock()
}

func (m *Manager) emitStatus(status string, dest string) {
	if m.events.Status == "" {
		return
	}
	payload := any(status)
	if m.events.StatusFactory != nil {
		payload = m.events.StatusFactory(status, dest)
	}
	application.Get().Event.Emit(m.events.Status, payload)
}

func (m *Manager) emitDone(dest string) {
	if m.events.Done == "" {
		return
	}
	payload := any(dest)
	if m.events.DoneFactory != nil {
		payload = m.events.DoneFactory(dest)
	}
	application.Get().Event.Emit(m.events.Done, payload)
}

func (m *Manager) emitError(err string, dest string) {
	if m.events.Error == "" {
		return
	}
	payload := any(err)
	if m.events.ErrorFactory != nil {
		payload = m.events.ErrorFactory(err, dest)
	}
	application.Get().Event.Emit(m.events.Error, payload)
}

func parseInt64(s string) (int64, error) {
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not int")
		}
		v = v*10 + int64(c-'0')
	}
	return v, nil
}
