package template

import "testing"

func TestOutputBaseNameOrMarkdownFallback(t *testing.T) {
	tests := []struct {
		name     string
		info     OutputInfo
		markdown string
		want     string
	}{
		{
			name: "uses template output basename",
			info: OutputInfo{OutputBaseName: "关于开展检查的通知"},
			want: "关于开展检查的通知",
		},
		{
			name: "skips generic output and uses preview title",
			info: OutputInfo{OutputBaseName: "output", PreviewTitle: "授课进度计划表 PLC"},
			want: "授课进度计划表 PLC",
		},
		{
			name:     "falls back to frontmatter title",
			info:     OutputInfo{OutputBaseName: "output"},
			markdown: "---\ntitle: \"安全生产专项检查\"\n---\n\n正文",
			want:     "安全生产专项检查",
		},
		{
			name:     "falls back to first h1",
			info:     OutputInfo{OutputBaseName: "untitled"},
			markdown: "# 电气设备控制线路安装与调试\n\n正文",
			want:     "电气设备控制线路安装与调试",
		},
		{
			name: "sanitizes unsafe title characters",
			info: OutputInfo{OutputBaseName: `a/b:c*?d"e<f>g|h`},
			want: "a_b_c__d_e_f_g_h",
		},
		{
			name: "uses stable product fallback for blank document",
			info: OutputInfo{OutputBaseName: "output"},
			want: "presto-document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OutputBaseNameOrMarkdownFallback(tt.info, tt.markdown)
			if got != tt.want {
				t.Fatalf("OutputBaseNameOrMarkdownFallback() = %q, want %q", got, tt.want)
			}
		})
	}
}
