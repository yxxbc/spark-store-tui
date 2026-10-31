package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func getJSON(ctx context.Context, client *http.Client, url string, target any) error {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "sparkstore/0.9")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("catalog request returned %s: %s", response.Status, string(body))
	}
	return json.NewDecoder(response.Body).Decode(target)
}
