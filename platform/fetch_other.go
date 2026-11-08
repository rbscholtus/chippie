//go:build !(js && wasm)

package platform

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Fetch downloads url with net/http. See fetch.go.
func Fetch(ctx context.Context, url string, limit int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %w: %d", url, ErrHTTPStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}

	if len(body) > limit {
		return nil, fmt.Errorf("fetch %s: %w", url, ErrTooLarge)
	}

	return body, nil
}
