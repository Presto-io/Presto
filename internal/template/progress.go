package template

import (
	"io"
)

// ProgressCallback is called with download progress updates
type ProgressCallback func(downloaded, total int64)

// ProgressReader wraps io.Reader and calls progress callback on each Read
type ProgressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress ProgressCallback
}

// NewProgressReader creates a new ProgressReader that wraps an io.Reader
// and calls onProgress after each Read operation with the current progress
func NewProgressReader(reader io.Reader, total int64, onProgress ProgressCallback) *ProgressReader {
	return &ProgressReader{
		reader:     reader,
		total:      total,
		onProgress: onProgress,
	}
}

// Read implements io.Reader. It reads from the underlying reader,
// tracks the number of bytes read, and calls the progress callback
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)
		if pr.onProgress != nil {
			pr.onProgress(pr.downloaded, pr.total)
		}
	}
	return n, err
}
