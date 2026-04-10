package template

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

// ProgressCallback is called with download progress updates
type ProgressCallback func(downloaded, total int64)

// ProgressReader wraps io.Reader and calls progress callback on each Read
type ProgressReader struct {
	reader      io.Reader
	total       int64
	downloaded  int64
	onProgress  ProgressCallback
	lastLog     time.Time     // NEW: 上次日志时间
	logInterval time.Duration // NEW: 日志间隔（默认 2s）
}

// NewProgressReader creates a new ProgressReader that wraps an io.Reader
// and calls onProgress after each Read operation with the current progress
func NewProgressReader(reader io.Reader, total int64, onProgress ProgressCallback) *ProgressReader {
	return &ProgressReader{
		reader:      reader,
		total:       total,
		onProgress:  onProgress,
		logInterval: 2 * time.Second, // 每 2 秒记录一次
	}
}

// Read implements io.Reader. It reads from the underlying reader,
// tracks the number of bytes read, and calls the progress callback
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)

		// 调用回调
		if pr.onProgress != nil {
			pr.onProgress(pr.downloaded, pr.total)
		}

		// 每 2 秒记录一次进度日志
		if time.Since(pr.lastLog) >= pr.logInterval {
			percent := float64(pr.downloaded) / float64(pr.total) * 100
			elapsed := time.Since(pr.lastLog).Seconds()
			var speed int64
			if elapsed > 0 {
				speed = pr.downloaded / int64(elapsed) // bytes/s
			}

			slog.Debug("[download] progress",
				"downloaded_bytes", pr.downloaded,
				"total_bytes", pr.total,
				"percent", fmt.Sprintf("%.1f%%", percent),
				"speed_bytes_per_sec", speed)

			pr.lastLog = time.Now()
		}
	}
	return n, err
}
