// internal/download/manager.go
package download

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"awesomeProject1/internal/util"
)

type Progress struct {
	BytesDone   int64
	BytesTotal  int64
	CurrentFile string
	ETA         string
	Done        bool
	Err         error
}

type Manager struct {
	client  *http.Client
	console *Console
}

func NewManager(console *Console) *Manager {
	// Tuned transport so we can hammer a single host efficiently.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		// no global Timeout: ROM sets can be huge; you can add per-request ctx later.
	}

	return &Manager{
		client:  client,
		console: console,
	}
}

// DownloadFileWithRetry wraps DownloadFile with simple retry logic.
func (m *Manager) DownloadFileWithRetry(urlStr, targetDir string, cb func(Progress), attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 1; i <= attempts; i++ {
		if m.console != nil && i > 1 {
			m.console.Log(fmt.Sprintf("Retry %d/%d for %s", i, attempts, urlStr))
		}
		lastErr = m.DownloadFile(urlStr, targetDir, cb)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

// DownloadFile downloads a single URL into targetDir and reports progress via cb.
// If the target .zip already exists, it will be skipped and still considered for extraction.
func (m *Manager) DownloadFile(urlStr, targetDir string, cb func(Progress)) error {
	start := time.Now()
	p := Progress{CurrentFile: urlStr}

	if cb == nil {
		cb = func(Progress) {}
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		p.Err = err
		cb(p)
		if m.console != nil {
			m.console.LogError(err.Error())
		}
		return err
	}

	// Determine filename from URL (simple approach; fine for Myrient)
	filename := filepath.Base(urlStr)
	if filename == "" || filename == "/" {
		filename = "download.bin"
	}
	dstPath := filepath.Join(targetDir, filename)

	// If file already exists, skip download but still attempt unzip
	if fi, err := os.Stat(dstPath); err == nil && fi.Size() > 0 {
		if m.console != nil {
			m.console.Log(fmt.Sprintf("Skipping existing file: %s", dstPath))
		}
		p.BytesTotal = fi.Size()
		p.BytesDone = fi.Size()
		p.Done = true
		cb(p)

		// Try to unzip existing file if it's a .zip
		return m.maybeUnzip(dstPath)
	}

	if m.console != nil {
		m.console.Log(fmt.Sprintf("Downloading %s -> %s", urlStr, dstPath))
	}

	resp, err := m.client.Get(urlStr)
	if err != nil {
		p.Err = err
		cb(p)
		if m.console != nil {
			m.console.LogError(err.Error())
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.Err = fmt.Errorf("http error: %s", resp.Status)
		cb(p)
		if m.console != nil {
			m.console.LogError(p.Err.Error())
		}
		return p.Err
	}

	total := resp.ContentLength
	p.BytesTotal = total

	out, err := os.Create(dstPath)
	if err != nil {
		p.Err = err
		cb(p)
		if m.console != nil {
			m.console.LogError(err.Error())
		}
		return err
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				p.Err = werr
				cb(p)
				if m.console != nil {
					m.console.LogError(werr.Error())
				}
				return werr
			}
			p.BytesDone += int64(n)
			p.ETA = util.CalculateETA(p.BytesDone, total, start)
			cb(p)
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			p.Err = rerr
			cb(p)
			if m.console != nil {
				m.console.LogError(rerr.Error())
			}
			return rerr
		}
	}

	p.Done = true
	cb(p)

	if m.console != nil {
		m.console.LogComplete()
		m.console.Log(fmt.Sprintf(
			"Downloaded %s (%s).",
			filename,
			util.FormatBytes(p.BytesDone, 2),
		))
	}

	// After successful download, unzip if needed
	return m.maybeUnzip(dstPath)
}

func (m *Manager) maybeUnzip(dstPath string) error {
	if !strings.HasSuffix(strings.ToLower(dstPath), ".zip") {
		return nil
	}
	if m.console != nil {
		m.console.Log("Extracting: " + dstPath)
	}
	outDir, err := util.UnzipZipFileInPlace(dstPath)
	if err != nil {
		if m.console != nil {
			m.console.LogError(fmt.Sprintf("Error extracting %s: %v", dstPath, err))
		}
		return err
	}
	if m.console != nil {
		m.console.Log("Extracted into directory: " + outDir)
	}
	return nil
}
