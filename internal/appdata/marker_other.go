//go:build !darwin && !linux

package appdata

func markPlatformGenerated(paths []string, marker Marker) {}
