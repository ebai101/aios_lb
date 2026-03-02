package proxy

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Race forwards an incoming request to multiple base URLs concurrently
// and returns the first successful HTTP 2xx response.
func Race(ctx context.Context, client *http.Client, req *http.Request, baseURLs []string, debug bool) (*http.Response, error) {
	resultCh := make(chan *http.Response, 1)
	allDone := make(chan struct{})
	var wg sync.WaitGroup

	for _, baseURL := range baseURLs {
		wg.Add(1)

		go func(bURL string) {
			defer wg.Done()

			outReq, err := cloneRequest(ctx, req, bURL)
			if err != nil {
				return
			}

			logDebug(debug, "OUTGOING Start -> %s", outReq.URL.String())

			reqStart := time.Now()
			resp, err := client.Do(outReq)
			reqDuration := time.Since(reqStart)

			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logDebug(debug, "OUTGOING Error -> %s (Err: %v, Time: %v)", outReq.URL.String(), err, reqDuration)
				}
				return
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				logDebug(debug, "OUTGOING Failed -> %s (Status: %d, Time: %v)", outReq.URL.String(), resp.StatusCode, reqDuration)
				if err := resp.Body.Close(); err != nil {
					log.Printf("[DEBUG] resp.Body.Close returned an error: %s", err)
				}
				return
			}

			select {
			case resultCh <- resp:
				logInfo("🏆 RACE WINNER -> %s (Time: %v)", outReq.URL.String(), reqDuration)
			default:
				if debug {
					logDebug(debug, "🛑 RACE LOSER -> %s (Time: %v)", outReq.URL.String(), reqDuration)
				}
				if err := resp.Body.Close(); err != nil {
					log.Printf("[DEBUG] resp.Body.Close returned an error: %s", err)
				}
			}
		}(baseURL)
	}

	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case resp := <-resultCh:
		return resp, nil
	case <-allDone:
		return nil, errors.New("all upstream instances failed or returned non-200")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func cloneRequest(ctx context.Context, originalReq *http.Request, baseURL string) (*http.Request, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	u.Path, err = url.JoinPath(u.Path, originalReq.URL.Path)
	if err != nil {
		return nil, err
	}

	u.RawQuery = originalReq.URL.RawQuery
	outReq, err := http.NewRequestWithContext(ctx, originalReq.Method, u.String(), originalReq.Body)
	if err != nil {
		return nil, err
	}
	outReq.Header = originalReq.Header.Clone()
	outReq.Host = ""

	return outReq, nil
}
