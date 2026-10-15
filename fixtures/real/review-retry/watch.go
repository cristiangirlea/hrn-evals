// Package watch follows a control service's job event stream, cut down from
// hrn's remote client.
package watch

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrMalformedEvent is an event the stream should never carry.
var ErrMalformedEvent = errors.New("malformed event from control service")

// ErrEventOutput is a failure to write an event to the caller's output.
var ErrEventOutput = errors.New("failed to write event output")

// Client sends its token only to the control service, never to a redirect.
type Client struct {
	origin string
	token  string
	http   *http.Client
}

// New returns a client for the control service at origin.
func New(origin, token string) *Client {
	return &Client{origin: strings.TrimRight(origin, "/"), token: token,
		http: &http.Client{Timeout: 20 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Watch follows a job's event stream at path, writing each event's JSON to out
// from the event after after. A dropped connection is retried every second,
// up to 30 failures in a row; it returns nil when ctx is done.
func (c *Client) Watch(ctx context.Context, path string, after int64, out io.Writer) error {
	watchClient := *c.http
	watchClient.Timeout = 0
	failures := 0
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.origin+path, nil)
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+c.token)
		request.Header.Set("Accept", "text/event-stream")
		request.Header.Set("Last-Event-ID", strconv.FormatInt(after, 10))
		response, err := watchClient.Do(request)
		if err == nil {
			if response.StatusCode != http.StatusOK {
				response.Body.Close()
				err = &HTTPError{Status: response.StatusCode}
				if response.StatusCode < 500 {
					return err
				}
			} else {
				failures = 0
				after, err = ReadEvents(response.Body, after, out)
				response.Body.Close()
				if ctx.Err() != nil {
					return nil
				}
				if errors.Is(err, ErrMalformedEvent) || errors.Is(err, ErrEventOutput) {
					return err
				}
			}
		} else {
			var unknown x509.UnknownAuthorityError
			var hostname x509.HostnameError
			var invalid x509.CertificateInvalidError
			if errors.As(err, &unknown) || errors.As(err, &hostname) || errors.As(err, &invalid) {
				return err
			}
		}
		if err != nil {
			failures++
			if failures >= 30 {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}

// ReadEvents writes the data of each event in input with an ID above after to
// out, one JSON value per line, and returns the last ID written. An event line
// may be at most 1 MiB.
func ReadEvents(input io.Reader, after int64, out io.Writer) (int64, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var currentID int64
	var data string
	emit := func() error {
		if data == "" {
			return nil
		}
		if currentID <= 0 {
			return fmt.Errorf("%w: missing positive ID", ErrMalformedEvent)
		}
		if currentID <= after {
			return nil
		}
		if !json.Valid([]byte(data)) {
			return fmt.Errorf("%w: invalid JSON", ErrMalformedEvent)
		}
		if _, err := fmt.Fprintln(out, data); err != nil {
			return fmt.Errorf("%w: %w", ErrEventOutput, err)
		}
		after = currentID
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := emit(); err != nil {
				return after, err
			}
			currentID, data = 0, ""
			continue
		}
		if value, ok := strings.CutPrefix(line, "id: "); ok {
			id, err := strconv.ParseInt(value, 10, 64)
			if err != nil || id < 0 {
				return after, fmt.Errorf("%w: invalid ID", ErrMalformedEvent)
			}
			currentID = id
		} else if value, ok := strings.CutPrefix(line, "data: "); ok {
			if data != "" {
				data += "\n"
			}
			data += value
		}
	}
	if err := scanner.Err(); err != nil {
		return after, err
	}
	if data != "" {
		if err := emit(); err != nil {
			return after, err
		}
	}
	return after, nil
}

// HTTPError omits untrusted error bodies, which could echo credentials.
type HTTPError struct{ Status int }

func (e *HTTPError) Error() string { return fmt.Sprintf("control service HTTP %d", e.Status) }
