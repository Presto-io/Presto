package template

import (
	"bytes"
	"io"
	"testing"
)

func TestProgressReader_Read(t *testing.T) {
	data := []byte("hello world")
	reader := bytes.NewReader(data)

	var callbackCalls int
	var lastDownloaded, lastTotal int64

	pr := NewProgressReader(reader, int64(len(data)), func(downloaded, total int64) {
		callbackCalls++
		lastDownloaded = downloaded
		lastTotal = total
	})

	buf := make([]byte, 5)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes read, got %d", n)
	}
	if pr.downloaded != 5 {
		t.Errorf("expected downloaded=5, got %d", pr.downloaded)
	}
	if callbackCalls == 0 {
		t.Error("callback was not called")
	}
	if lastDownloaded != 5 {
		t.Errorf("callback received downloaded=%d, expected 5", lastDownloaded)
	}
	if lastTotal != int64(len(data)) {
		t.Errorf("callback received total=%d, expected %d", lastTotal, len(data))
	}
}

func TestProgressReader_Callback(t *testing.T) {
	data := []byte("test data for progress tracking")
	reader := bytes.NewReader(data)

	var callbacks []struct {
		downloaded int64
		total      int64
	}

	pr := NewProgressReader(reader, int64(len(data)), func(downloaded, total int64) {
		callbacks = append(callbacks, struct {
			downloaded int64
			total      int64
		}{downloaded, total})
	})

	// Read in chunks
	buf := make([]byte, 10)
	for {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}

	// Verify multiple callbacks were made
	if len(callbacks) < 2 {
		t.Errorf("expected at least 2 callbacks, got %d", len(callbacks))
	}

	// Verify total is consistent
	for i, cb := range callbacks {
		if cb.total != int64(len(data)) {
			t.Errorf("callback[%d]: total=%d, expected %d", i, cb.total, len(data))
		}
	}

	// Verify downloaded increases
	for i := 1; i < len(callbacks); i++ {
		if callbacks[i].downloaded < callbacks[i-1].downloaded {
			t.Errorf("callback[%d]: downloaded decreased from %d to %d",
				i, callbacks[i-1].downloaded, callbacks[i].downloaded)
		}
	}
}

func TestProgressReader_Total(t *testing.T) {
	data := []byte("short")
	reader := bytes.NewReader(data)

	var receivedTotal int64 = -1
	pr := NewProgressReader(reader, 1000, func(downloaded, total int64) {
		receivedTotal = total
	})

	buf := make([]byte, len(data))
	_, err := pr.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}

	if receivedTotal != 1000 {
		t.Errorf("expected total=1000, got %d", receivedTotal)
	}
}
