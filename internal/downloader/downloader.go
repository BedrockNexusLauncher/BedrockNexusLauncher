package downloader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	// Immutable after construction; safe to read from any goroutine.
	ctx         context.Context
	url         string
	dest        string
	expectedMD5 string
	verify      func(string) error

	// Mutable task state. The worker goroutine and the Manager control
	// methods (Pause/Resume/Cancel/…) touch these concurrently, so they
	// are atomics — previously they were plain fields written under
	// Manager.mu by the control methods but read/written unguarded by
	// the worker, a data race (M-5). Event emission never happens
	// while holding a lock; it reads plain snapshots of these values.
	retryCount atomic.Int32
	total      atomic.Int64
	downloaded atomic.Int64
	paused     atomic.Bool
	cancelled  atomic.Bool
	running    atomic.Bool
	// active is a CAS guard so two run() goroutines never work on the
	// same task at once (e.g. racing Resume calls).
	active atomic.Bool

	// cancelFn is guarded by Manager.mu.
	cancelFn context.CancelFunc
}

func NewManager(events Events, opts Options) *Manager {
	if opts.Throttle <= 0 {
		opts.Throttle = 250 * time.Millisecond
	}
	return &Manager{events: events, opts: opts, tasks: map[string]*state{}}
}

func (m *Manager) Start(ctx context.Context, src string, dest string, md5sum string) string {
	return m.start(ctx, src, dest, md5sum, nil)
}

// StartVerified publishes the final file only after a package-specific integrity
// check. Progress and cancellation retain the same destination and event shape.
func (m *Manager) StartVerified(ctx context.Context, src, dest string, verify func(string) error) string {
	return m.start(ctx, src, dest, "", verify)
}

func (m *Manager) start(ctx context.Context, src string, dest string, md5sum string, verify func(string) error) string {
	dir := filepath.Dir(dest)
	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	m.mu.Lock()
	if m.tasks == nil {
		m.tasks = map[string]*state{}
	}
	if existing, ok := m.tasks[dest]; ok && existing != nil && existing.running.Load() {
		m.mu.Unlock()
		m.emitStatus("started", dest)
		return dest
	}
	local := &state{ctx: ctx, url: src, dest: dest, expectedMD5: md5sum, verify: verify}
	m.tasks[dest] = local
	m.mu.Unlock()
	go m.run(local)
	m.emitStatus("started", dest)
	return dest
}

// The control methods below mutate task state under Manager.mu (the
// tasks map and cancelFn) and the atomic fields, collect the events to
// publish, and only emit after unlocking — event emission must never
// happen while holding the lock (M-5).

func (m *Manager) Pause() {
	m.mu.Lock()
	var dests []string
	for _, st := range m.tasks {
		if st == nil {
			continue
		}
		st.paused.Store(true)
		if st.cancelFn != nil {
			st.cancelFn()
		}
		dests = append(dests, st.dest)
	}
	m.mu.Unlock()
	for _, dest := range dests {
		m.emitStatus("paused", dest)
	}
}

func (m *Manager) Resume() {
	m.mu.Lock()
	var resumed []*state
	for _, st := range m.tasks {
		if st == nil || st.cancelled.Load() {
			continue
		}
		if st.running.Load() {
			continue
		}
		st.paused.Store(false)
		resumed = append(resumed, st)
	}
	m.mu.Unlock()
	for _, st := range resumed {
		go m.run(st)
		m.emitStatus("resumed", st.dest)
	}
}

func (m *Manager) Cancel() {
	m.mu.Lock()
	var dests []string
	for dest, st := range m.tasks {
		if st == nil {
			continue
		}
		st.cancelled.Store(true)
		if st.cancelFn != nil {
			st.cancelFn()
		}
		if !st.running.Load() {
			delete(m.tasks, dest)
		}
		dests = append(dests, st.dest)
	}
	m.mu.Unlock()
	for _, dest := range dests {
		m.emitStatus("cancelled", dest)
	}
}

func (m *Manager) PauseTask(dest string) {
	m.mu.Lock()
	st, ok := m.tasks[dest]
	if ok && st != nil {
		st.paused.Store(true)
		if st.cancelFn != nil {
			st.cancelFn()
		}
	}
	m.mu.Unlock()
	if ok && st != nil {
		m.emitStatus("paused", st.dest)
	}
}

func (m *Manager) ResumeTask(dest string) {
	m.mu.Lock()
	st, ok := m.tasks[dest]
	resume := ok && st != nil && !st.cancelled.Load() && !st.running.Load()
	if resume {
		st.paused.Store(false)
	}
	m.mu.Unlock()
	if resume {
		go m.run(st)
		m.emitStatus("resumed", st.dest)
	}
}

func (m *Manager) CancelTask(dest string) {
	m.mu.Lock()
	st, ok := m.tasks[dest]
	if ok && st != nil {
		st.cancelled.Store(true)
		if st.cancelFn != nil {
			st.cancelFn()
		}
		if !st.running.Load() {
			delete(m.tasks, dest)
		}
	}
	m.mu.Unlock()
	if ok && st != nil {
		m.emitStatus("cancelled", st.dest)
	}
}

func (m *Manager) run(s *state) {
	if !s.active.CompareAndSwap(false, true) {
		return
	}
	defer s.active.Store(false)

	m.mu.Lock()
	local := m.tasks[s.dest]
	if local != s {
		m.mu.Unlock()
		return
	}
	if local == nil || local.cancelled.Load() || local.paused.Load() {
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
		local.downloaded.Store(cur)
		ctx, cancel := context.WithCancel(local.ctx)
		m.mu.Lock()
		local.cancelFn = cancel
		m.tasks[local.dest] = local
		m.mu.Unlock()
		local.running.Store(true)

		req, err := http.NewRequestWithContext(ctx, "GET", local.url, nil)
		if err != nil {
			cancel()
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
			cancel()
			if !m.retry(local, attempt, err) {
				m.emitError(err.Error(), local.dest)
				m.finishRunning(local)
				return
			}
			continue
		}

		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && cur > 0 {
			resp.Body.Close()
			cancel()
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
			cancel()
			statusErr := fmt.Errorf("HTTP %s", resp.Status)
			// Transient statuses (429 rate limiting, 5xx server
			// errors) deserve the same retry budget/backoff as
			// transport errors; other 4xx are deterministic
			// request failures and stay fatal.
			if isRetryableStatus(resp.StatusCode) && m.retry(local, attempt, statusErr) {
				continue
			}
			m.emitError(statusErr.Error(), local.dest)
			m.finishRunning(local)
			return
		}

		if m.opts.Resume && cur > 0 && resp.StatusCode == http.StatusOK {
			_ = os.Remove(downloadDest)
			cur = 0
			metadata = partialMetadata{URL: local.url}
			local.downloaded.Store(0)
		}

		// On a resumed download the server must continue exactly where
		// the local partial ends. A 206 whose Content-Range starts
		// anywhere else (or cannot be parsed) would silently splice two
		// different byte streams into one file (M-2): discard the
		// partial and restart from zero instead.
		if resp.StatusCode == http.StatusPartialContent {
			rangeStart, _, _, rangeOK := parseContentRange(resp.Header.Get("Content-Range"))
			if !rangeOK || rangeStart != cur {
				resp.Body.Close()
				cancel()
				_ = os.Remove(downloadDest)
				_ = os.Remove(metadataDest)
				if attempt < maxDownloadRetries {
					continue
				}
				m.emitError("ERR_INVALID_CONTENT_RANGE", local.dest)
				m.finishRunning(local)
				return
			}
		}
		metadata.URL = local.url
		metadata.ETag = resp.Header.Get("ETag")
		metadata.LastModified = resp.Header.Get("Last-Modified")

		total := resp.ContentLength
		if resp.StatusCode == http.StatusPartialContent {
			if _, _, rangeTotal, rangeOK := parseContentRange(resp.Header.Get("Content-Range")); rangeOK && rangeTotal >= 0 {
				total = rangeTotal
			} else if total > 0 {
				total = cur + total
			}
		}
		if total <= 0 && metadata.Total > 0 {
			total = metadata.Total
		}
		metadata.Total = total
		if m.opts.Resume {
			if err := writeMetadata(metadataDest, metadata); err != nil {
				// Non-fatal (a later attempt rewrites it), but a
				// silently dropped write would surface as a lost
				// resume position, so it must not go unlogged.
				log.Printf("downloader: writing resume metadata for %s: %v", local.dest, err)
			}
		}
		local.total.Store(total)

		flags := os.O_CREATE | os.O_WRONLY
		if cur == 0 {
			flags |= os.O_TRUNC
		} else {
			flags |= os.O_APPEND
		}
		f, err := os.OpenFile(downloadDest, flags, 0o644)
		if err != nil {
			resp.Body.Close()
			cancel()
			m.emitError(err.Error(), local.dest)
			m.finishRunning(local)
			return
		}
		{
			snap := DownloadProgress{Downloaded: local.downloaded.Load(), Total: local.total.Load(), Dest: local.dest}
			payload := any(snap)
			if m.events.ProgressFactory != nil {
				payload = m.events.ProgressFactory(snap)
			}
			application.Get().Event.Emit(m.events.Progress, payload)
		}
		buf := make([]byte, 128*1024)
		lastEmit := time.Now()

		// Stall watchdog (M-1): a server that trickles or stops sending
		// without closing the connection must not hang the attempt
		// forever. The watchdog cancels the attempt context when no
		// body bytes have arrived for stallTimeout; it is always joined
		// (stop) before this attempt continues or returns.
		watchdog := startStallWatchdog(cancel, stallTimeout)

		var loopErr error
		for {
			if local.cancelled.Load() || local.paused.Load() {
				if err := f.Sync(); err != nil {
					log.Printf("downloader: syncing partial file for %s: %v", local.dest, err)
				}
				_ = f.Close()
				resp.Body.Close()
				watchdog.stop()
				cancelled := local.cancelled.Load()
				if cancelled && m.opts.RemoveOnCancel && downloadDest != "" {
					_ = os.Remove(downloadDest)
				}
				m.finishRunning(local)
				if cancelled {
					m.emitStatus("cancelled", local.dest)
				}
				cancel()
				return
			}
			n, er := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.Write(buf[:n]); werr != nil {
					loopErr = werr
					break
				}
				watchdog.touch()
				local.downloaded.Add(int64(n))
				if time.Since(lastEmit) >= m.opts.Throttle {
					snap := DownloadProgress{Downloaded: local.downloaded.Load(), Total: local.total.Load(), Dest: local.dest}
					payload := any(snap)
					if m.events.ProgressFactory != nil {
						payload = m.events.ProgressFactory(snap)
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
			if err := f.Sync(); err != nil {
				log.Printf("downloader: syncing partial file for %s: %v", local.dest, err)
			}
		}
		_ = f.Close()
		resp.Body.Close()
		watchdog.stop()

		if loopErr != nil {
			// Pause and cancel both abort the in-flight read by
			// cancelling the attempt context, so the resulting read
			// error lands here — they must not be conflated (M-6):
			// a paused task keeps its partial file (Resume needs it)
			// even when RemoveOnCancel is set, and reports "paused",
			// never "cancelled".
			cancelled := local.cancelled.Load()
			paused := local.paused.Load()
			switch {
			case cancelled:
				if m.opts.RemoveOnCancel && downloadDest != "" {
					_ = os.Remove(downloadDest)
				}
				m.emitStatus("cancelled", local.dest)
			case paused:
				m.emitStatus("paused", local.dest)
			case watchdog.fired.Load():
				stallErr := fmt.Errorf("download stalled: no progress for %s", stallTimeout)
				if m.retry(local, attempt, stallErr) {
					cancel()
					continue
				}
				m.emitError(stallErr.Error(), local.dest)
			case ctx.Err() == context.Canceled:
				if m.opts.RemoveOnCancel && downloadDest != "" {
					_ = os.Remove(downloadDest)
				}
				m.emitStatus("cancelled", local.dest)
			default:
				if m.retry(local, attempt, loopErr) {
					cancel()
					continue
				}
				m.emitError(loopErr.Error(), local.dest)
			}
			m.finishRunning(local)
			cancel()
			return
		}

		// The transfer ended cleanly, but a clean EOF before the known
		// total still means the file is incomplete (truncating server
		// or proxy): never publish a short file as if it were whole
		// (M-2). The partial stays on disk so a retry resumes from it.
		if want := local.total.Load(); want > 0 && local.downloaded.Load() != want {
			shortErr := fmt.Errorf("incomplete download: got %d of %d bytes", local.downloaded.Load(), want)
			if m.retry(local, attempt, shortErr) {
				cancel()
				continue
			}
			m.emitError(shortErr.Error(), local.dest)
			m.finishRunning(local)
			cancel()
			return
		}

		// Download complete, verify MD5
		if local.expectedMD5 != "" {
			m.emitStatus("verifying", local.dest)
			if hash, err := calculateMD5(downloadDest); err == nil {
				if !strings.EqualFold(hash, local.expectedMD5) {
					// Mismatch
					_ = os.Remove(downloadDest)
					count := local.retryCount.Add(1)
					if count < 3 {
						m.emitError(fmt.Sprintf("MD5 mismatch (attempt %d/3), retrying...", count), local.dest)
						m.emitStatus("started", local.dest)
						time.Sleep(1 * time.Second)
						cancel()
						continue
					} else {
						m.emitError("ERR_MD5_MISMATCH", local.dest)
						m.finishRunning(local)
						cancel()
						return
					}
				}
			} else {
				// Failed to calculate MD5, maybe treat as error?
				m.emitError(fmt.Sprintf("MD5 calculation failed: %v", err), local.dest)
				m.finishRunning(local)
				cancel()
				return
			}
		}

		if local.verify != nil {
			m.emitStatus("verifying", local.dest)
			if err := local.verify(downloadDest); err != nil {
				_ = os.Remove(downloadDest)
				m.emitError(err.Error(), local.dest)
				m.finishRunning(local)
				cancel()
				return
			}
		}

		if err := os.Rename(downloadDest, local.dest); err != nil {
			m.emitError(err.Error(), local.dest)
		} else {
			_ = os.Remove(metadataDest)
			snap := DownloadProgress{Downloaded: local.downloaded.Load(), Total: local.total.Load(), Dest: local.dest}
			payload := any(snap)
			if m.events.ProgressFactory != nil {
				payload = m.events.ProgressFactory(snap)
			}
			application.Get().Event.Emit(m.events.Progress, payload)
			m.emitDone(local.dest)
		}
		m.finishRunning(local)
		cancel()
		return
	}
}

// isRetryableStatus reports whether an HTTP response status is
// transient enough to be worth retrying: 429 (rate limited) and any
// 5xx (server-side failure). Other 4xx describe the request itself
// and would fail identically on every attempt.
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

func (m *Manager) retry(s *state, attempt int, err error) bool {
	if attempt >= maxDownloadRetries || s.cancelled.Load() || s.paused.Load() {
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
	s.running.Store(false)
	if s.cancelled.Load() || !s.paused.Load() {
		m.mu.Lock()
		if m.tasks != nil {
			if cur, ok := m.tasks[s.dest]; ok && cur == s {
				delete(m.tasks, s.dest)
			}
		}
		m.mu.Unlock()
	}
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

// stallTimeout is how long a download attempt may go without a single
// body byte arriving before the stall watchdog aborts it (M-1).
const stallTimeout = 60 * time.Second

// stallWatchdog aborts a download attempt whose body makes no
// progress. It runs one small timer goroutine per attempt; stop()
// joins that goroutine, so it never outlives its attempt.
type stallWatchdog struct {
	lastProgress atomic.Int64 // UnixNano of the last successful body read
	fired        atomic.Bool
	cancel       context.CancelFunc
	stopCh       chan struct{}
	doneCh       chan struct{}
	stopOnce     sync.Once
}

func startStallWatchdog(cancel context.CancelFunc, timeout time.Duration) *stallWatchdog {
	w := &stallWatchdog{
		cancel: cancel,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	w.touch()
	go w.watch(timeout)
	return w
}

func (w *stallWatchdog) watch(timeout time.Duration) {
	defer close(w.doneCh)
	interval := timeout / 6
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case now := <-ticker.C:
			if now.Sub(time.Unix(0, w.lastProgress.Load())) >= timeout {
				w.fired.Store(true)
				w.cancel()
				return
			}
		}
	}
}

// touch records body progress; called on every successful read.
func (w *stallWatchdog) touch() {
	w.lastProgress.Store(time.Now().UnixNano())
}

// stop terminates the watchdog goroutine and waits for it to exit.
func (w *stallWatchdog) stop() {
	w.stopOnce.Do(func() { close(w.stopCh) })
	<-w.doneCh
}

// parseContentRange parses an HTTP Content-Range header value of the
// form "bytes <start>-<end>/<total>" (total may be "*"). total is -1
// when the server reports "*". ok is false for anything malformed —
// the caller must treat an unparseable range on a resumed download as
// a reason to restart from zero, never to splice blindly (M-2).
func parseContentRange(header string) (start, end, total int64, ok bool) {
	rest, found := strings.CutPrefix(strings.TrimSpace(header), "bytes ")
	if !found {
		return 0, 0, 0, false
	}
	rangePart, totalPart, found := strings.Cut(rest, "/")
	if !found {
		return 0, 0, 0, false
	}
	startPart, endPart, found := strings.Cut(rangePart, "-")
	if !found || startPart == "" || endPart == "" {
		return 0, 0, 0, false
	}
	start, err := parseInt64(startPart)
	if err != nil {
		return 0, 0, 0, false
	}
	end, err = parseInt64(endPart)
	if err != nil {
		return 0, 0, 0, false
	}
	total = -1
	if tp := strings.TrimSpace(totalPart); tp == "" {
		return 0, 0, 0, false
	} else if tp != "*" {
		v, err := parseInt64(tp)
		if err != nil {
			return 0, 0, 0, false
		}
		total = v
	}
	return start, end, total, true
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
