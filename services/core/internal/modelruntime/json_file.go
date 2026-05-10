package modelruntime

import (
	"fmt"
	"io"
	"os"
)

const maxRuntimeJSONFileBytes = 8 << 20

func readRuntimeJSONFile(path, label string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxRuntimeJSONFileBytes {
		return nil, fmt.Errorf("%s too large: %d bytes exceeds %d byte limit", label, info.Size(), maxRuntimeJSONFileBytes)
	}
	body, err := io.ReadAll(io.LimitReader(f, maxRuntimeJSONFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxRuntimeJSONFileBytes {
		return nil, fmt.Errorf("%s too large: exceeds %d byte limit", label, maxRuntimeJSONFileBytes)
	}
	return body, nil
}
