package cli

import (
	"bytes"
	"io"
	"os"
	goRuntime "runtime"
)

func readPipedInput() io.Reader {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return bytes.NewReader(data)
}

func runtimeVersion() string {
	return goRuntime.Version()
}

func runtimeOS() string {
	return goRuntime.GOOS
}

func runtimeArch() string {
	return goRuntime.GOARCH
}
