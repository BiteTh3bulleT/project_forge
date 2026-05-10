package gateway

import (
	"fmt"
	"io"
)

const (
	gatewayNetFetchResponseBodyLimit       int64 = 256 << 10
	gatewayWebSearchResponseBodyLimit      int64 = 512 << 10
	gatewayCapabilityHTTPResponseBodyLimit int64 = 512 << 10
	gatewayDesktopBridgeResponseBodyLimit  int64 = 2 << 20
)

func readGatewayHTTPResponseBody(body io.Reader, label string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s response limit must be positive", label)
	}
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s response too large: exceeds %d byte limit", label, limit)
	}
	return data, nil
}
