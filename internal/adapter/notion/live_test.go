package notion

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNotionLiveSmokeWithRealToken(t *testing.T) {
	// 真实 Notion 联调默认关闭，只有显式声明后才运行，避免普通单测污染真实工作区。
	if strings.TrimSpace(os.Getenv("CLAWRISE_RUN_NOTION_LIVE")) != "1" {
		t.Skip("未设置 CLAWRISE_RUN_NOTION_LIVE=1，跳过真实 Notion 联调")
	}
	if testing.Short() {
		t.Skip("short 模式下跳过真实 Notion 联调")
	}

	for _, name := range []string{"NOTION_TOKEN", "NOTION_PARENT_PAGE_ID"} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			t.Fatalf("缺少真实联调所需环境变量 %s", name)
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位当前测试文件路径")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	scriptPath := filepath.Join(repoRoot, "scripts", "ci", "run-notion-live.sh")

	command := exec.Command("bash", scriptPath)
	command.Dir = repoRoot
	command.Env = os.Environ()

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("真实 Notion 联调失败: %v\n%s", err, string(output))
	}

	t.Logf("真实 Notion 联调通过:\n%s", string(output))
}
