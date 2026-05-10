package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	urlFlag      = flag.String("url", "http://localhost:8080", "Target server URL")
	workersFlag  = flag.Int("workers", 200, "Number of concurrent workers")
	durationFlag = flag.Duration("duration", 10*time.Second, "Duration of the load test")
	hotItemFlag  = flag.Bool("hot", false, "Simulate a hot auction (all bids on 1 item)")
	itemsFlag    = flag.Int("items", 100, "Number of items to distribute load across")
	distFlag     = flag.String("dist", "uniform", "Distribution pattern: uniform or zipf")
)

type metrics struct {
	totalRequests  int64
	successPost    int64
	successGet     int64
	failedRequests int64
	totalLatencyMs int64
}

func main() {
	flag.Parse()

	fmt.Printf("==> Starting Load Test\n")
	fmt.Printf("URL:      %s\n", *urlFlag)
	fmt.Printf("Workers:  %d\n", *workersFlag)
	fmt.Printf("Duration: %v\n", *durationFlag)
	fmt.Printf("Hot Item: %v\n\n", *hotItemFlag)

	var stats metrics
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			MaxConnsPerHost:     1000,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
		},
	}

	startTime := time.Now()
	deadline := startTime.Add(*durationFlag)

	var lastErr atomic.Value
	for i := 0; i < *workersFlag; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			var zipf *rand.Zipf
			if *distFlag == "zipf" {
				// s > 1.1, q >= 1. s=1.1 means very skewed, s=2.0 means less skewed.
				zipf = rand.NewZipf(r, 1.1, 1, uint64(*itemsFlag-1))
			}

			for time.Now().Before(deadline) {
				itemID := "item_1"
				if *hotItemFlag {
					itemID = "item_hot"
				} else if *distFlag == "zipf" {
					itemID = fmt.Sprintf("item_%d", zipf.Uint64())
				} else {
					itemID = fmt.Sprintf("item_%d", r.Intn(*itemsFlag))
				}
				userID := fmt.Sprintf("user_%d", r.Intn(10000)) // Scaled user base
				amount := float64(r.Intn(1000)) + 1.0

				startReq := time.Now()
				var err error
				var statusCode int

				isPost := r.Float32() < 0.7
				// Increased retries for high RPS network glitches
				for attempt := 0; attempt < 5; attempt++ {
					if isPost {
						statusCode, err = postBid(client, itemID, userID, amount)
					} else {
						statusCode, err = getWinningBid(client, itemID)
					}
					if err == nil {
						break
					}
					// Small backoff if we fail (Senior approach)
					time.Sleep(time.Duration(attempt) * time.Millisecond)
				}

				atomic.AddInt64(&stats.totalRequests, 1)
				atomic.AddInt64(&stats.totalLatencyMs, time.Since(startReq).Milliseconds())

				if err != nil {
					atomic.AddInt64(&stats.failedRequests, 1)
					lastErr.Store(err.Error())
				} else if statusCode >= 400 && statusCode != http.StatusConflict && statusCode != http.StatusNotFound {
					atomic.AddInt64(&stats.failedRequests, 1)
					lastErr.Store(fmt.Sprintf("HTTP %d", statusCode))
				} else {
					if isPost {
						atomic.AddInt64(&stats.successPost, 1)
					} else {
						atomic.AddInt64(&stats.successGet, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	errMsg := ""
	if v := lastErr.Load(); v != nil {
		errMsg = v.(string)
	}
	printReport(&stats, elapsed, errMsg)
}

func postBid(client *http.Client, itemID, userID string, amount float64) (int, error) {
	payload := map[string]interface{}{
		"item_id": itemID,
		"user_id": userID,
		"amount":  amount,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, *urlFlag+"/bids", bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

func getWinningBid(client *http.Client, itemID string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/items/%s/winning-bid", *urlFlag, itemID), nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

func printReport(stats *metrics, elapsed time.Duration, lastErr string) {
	fmt.Println("==> Load Test Completed")
	fmt.Printf("Time Taken:         %v\n", elapsed)
	fmt.Printf("Total Requests:     %d\n", stats.totalRequests)
	fmt.Printf("Successful POSTs:   %d\n", stats.successPost)
	fmt.Printf("Successful GETs:    %d\n", stats.successGet)
	fmt.Printf("Failed Requests:    %d\n", stats.failedRequests)

	if lastErr != "" {
		fmt.Printf("Last Error:         %s\n", lastErr)
	}

	rps := float64(stats.totalRequests) / elapsed.Seconds()
	avgLatency := float64(stats.totalLatencyMs) / float64(stats.totalRequests)

	fmt.Println("--------------------------------------------------")
	fmt.Printf("Throughput (RPS):   %.2f requests/second\n", rps)
	fmt.Printf("Average Latency:    %.2f ms\n", avgLatency)
	fmt.Println("--------------------------------------------------")
}
