package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/theabys/enerflux/internal/measurements"
)

type HTTPUploader struct {
	client   *http.Client
	endpoint string
}

func NewHTTPUploader(endpoint string) *HTTPUploader {
	return &HTTPUploader{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		endpoint: strings.TrimRight(endpoint, "/"),
	}
}

func (u *HTTPUploader) Upload(
	ctx context.Context,
	items []measurements.MeasurementPayload,
) error {
	if len(items) == 0 {
		return nil
	}

	payload, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal measurements: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.endpoint+"/api/v1/measurements",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload measurements: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf(
				"upload measurements: unexpected status %s",
				resp.Status,
			)
		}

		return fmt.Errorf(
			"upload measurements: unexpected status %s: %s",
			resp.Status,
			strings.TrimSpace(string(body)),
		)
	}

	return nil
}

func ToPayloads(items []measurements.Measurement) []measurements.MeasurementPayload {
	payloads := make([]measurements.MeasurementPayload, len(items))

	for i := range items {
		payloads[i] = measurements.MeasurementPayload(items[i])
	}

	return payloads
}
