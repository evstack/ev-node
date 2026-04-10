package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type serverStats struct {
	ExecutedTxs    uint64 `json:"executed_txs"`
	BlocksProduced uint64 `json:"blocks_produced"`
}

const uniquePoolSize = 65536

var uniqueBodies [uniquePoolSize]string

func init() {
	for i := range uniquePoolSize {
		uniqueBodies[i] = "k" + strconv.Itoa(i) + "=v"
	}
}

func main() {
	addr := flag.String("addr", "localhost:9090", "server host:port")
	duration := flag.Duration("duration", 10*time.Second, "test duration")
	workers := flag.Int("workers", 1000, "concurrent workers")
	batchSize := flag.Int("batch", 1, "transactions per request (uses /tx/batch when >1)")
	targetRPS := flag.Uint64("target-rps", 1_000_000, "target transactions per second")
	flag.Parse()

	fmt.Printf("Stress Test Configuration\n")
	fmt.Printf("  Server:    %s\n", *addr)
	fmt.Printf("  Duration:  %s\n", *duration)
	fmt.Printf("  Workers:   %d\n", *workers)
	fmt.Printf("  Batch:     %d txs/request\n", *batchSize)
	fmt.Printf("  Goal:      %d tx/s\n\n", *targetRPS)

	if err := checkServer(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Server check failed: %v\n", err)
		os.Exit(1)
	}

	before := fetchStats(*addr)

	var success atomic.Uint64
	var failures atomic.Uint64
	var totalTxs atomic.Uint64

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	done := make(chan struct{}, *workers)
	for i := 0; i < *workers; i++ {
		if *batchSize > 1 {
			go batchWorker(ctx, *addr, *batchSize, &success, &failures, &totalTxs, done)
		} else {
			go singleWorker(ctx, *addr, &success, &failures, &totalTxs, done)
		}
	}

	start := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastTxs uint64
	var peakTPS uint64

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			cur := totalTxs.Load()
			tps := cur - lastTxs
			lastTxs = cur
			if tps > peakTPS {
				peakTPS = tps
			}
			elapsed := time.Since(start).Truncate(time.Second)
			fmt.Printf("\r[%6s] Total txs: %12d | Success: %12d | Fail: %8d | TPS: %12d   ",
				elapsed, cur, success.Load(), failures.Load(), tps)
		}
	}

	elapsed := time.Since(start)

	for i := 0; i < *workers; i++ {
		<-done
	}

	after := fetchStats(*addr)

	total := totalTxs.Load()
	avgTPS := float64(total) / elapsed.Seconds()

	var txsPerBlock float64
	deltaBlocks := after.BlocksProduced - before.BlocksProduced
	deltaTxs := after.ExecutedTxs - before.ExecutedTxs
	if deltaBlocks > 0 {
		txsPerBlock = float64(deltaTxs) / float64(deltaBlocks)
	}

	reached := avgTPS >= float64(*targetRPS)

	fmt.Println()
	fmt.Println()
	printResults(elapsed, uint64(*workers), uint64(*batchSize), total, success.Load(), failures.Load(),
		avgTPS, float64(peakTPS), deltaBlocks, deltaTxs, txsPerBlock, reached, *targetRPS)
}

func singleWorker(ctx context.Context, addr string, success, failures, totalTxs *atomic.Uint64, done chan struct{}) {
	defer func() { done <- struct{}{} }()

	rawReqs := make([][]byte, uniquePoolSize)
	for i := range uniquePoolSize {
		body := uniqueBodies[i]
		rawReqs[i] = fmt.Appendf(nil,
			"POST /tx HTTP/1.1\r\nHost: %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			addr, len(body), body,
		)
	}

	var idx uint64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			failures.Add(1)
			continue
		}

		br := bufio.NewReaderSize(conn, 512)

		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			default:
			}

			i := atomic.AddUint64(&idx, 1) % uniquePoolSize
			if _, err := conn.Write(rawReqs[i]); err != nil {
				failures.Add(1)
				conn.Close()
				break
			}

			resp, err := http.ReadResponse(br, nil)
			if err != nil {
				failures.Add(1)
				conn.Close()
				break
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusAccepted {
				success.Add(1)
				totalTxs.Add(1)
			} else {
				failures.Add(1)
			}
		}
	}
}

func batchWorker(ctx context.Context, addr string, batchSize int, success, failures, totalTxs *atomic.Uint64, done chan struct{}) {
	defer func() { done <- struct{}{} }()

	var idx uint64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			failures.Add(1)
			continue
		}

		br := bufio.NewReaderSize(conn, 4096)

		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			default:
			}

			base := atomic.AddUint64(&idx, uint64(batchSize))
			var body strings.Builder
			body.Grow(batchSize * 10)
			for i := range batchSize {
				body.WriteString(uniqueBodies[(base+uint64(i))%uniquePoolSize])
				body.WriteByte('\n')
			}
			bodyStr := body.String()

			header := fmt.Sprintf(
				"POST /tx/batch HTTP/1.1\r\nHost: %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n",
				addr, len(bodyStr),
			)
			rawReq := []byte(header + bodyStr)

			if _, err := conn.Write(rawReq); err != nil {
				failures.Add(1)
				conn.Close()
				break
			}

			resp, err := http.ReadResponse(br, nil)
			if err != nil {
				failures.Add(1)
				conn.Close()
				break
			}
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
				var result struct {
					Accepted int `json:"accepted"`
				}
				json.Unmarshal(respBody, &result)
				success.Add(1)
				totalTxs.Add(uint64(result.Accepted))
			} else {
				failures.Add(1)
			}
		}
	}
}

func checkServer(addr string) error {
	resp, err := http.Get("http://" + addr + "/store")
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", addr, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

func fetchStats(addr string) serverStats {
	resp, err := http.Get("http://" + addr + "/stats")
	if err != nil {
		return serverStats{}
	}
	defer resp.Body.Close()
	var s serverStats
	json.NewDecoder(resp.Body).Decode(&s)
	return s
}

func formatNum(n uint64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteString(string(c))
	}
	return result.String()
}

func printResults(elapsed time.Duration, workers, batchSize, total, success, failures uint64,
	avgTPS, peakTPS float64, blocks, executedTxs uint64, txsPerBlock float64, reached bool, targetRPS uint64) {

	goalLabel := fmt.Sprintf("Goal (%s tx/s)", formatNum(targetRPS))
	sep := "+----------------------------------------+----------------------------------------+"
	rowFmt := "| %-38s | %-38s |"

	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "STRESS TEST RESULTS", "")
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Duration", elapsed.Truncate(time.Millisecond).String())
	fmt.Printf(rowFmt+"\n", "Workers", fmt.Sprintf("%d", workers))
	fmt.Printf(rowFmt+"\n", "Batch Size", fmt.Sprintf("%d txs/request", batchSize))
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Total Transactions", formatNum(total))
	fmt.Printf(rowFmt+"\n", "Successful Requests", formatNum(success))
	fmt.Printf(rowFmt+"\n", "Failed Requests", formatNum(failures))
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Avg tx/s", formatFloat(avgTPS))
	fmt.Printf(rowFmt+"\n", "Peak tx/s (1s window)", formatFloat(peakTPS))
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Server Blocks Produced", formatNum(blocks))
	fmt.Printf(rowFmt+"\n", "Server Txs Executed", formatNum(executedTxs))
	fmt.Printf(rowFmt+"\n", "Avg Txs per Block", fmt.Sprintf("%.2f", txsPerBlock))
	fmt.Println(sep)

	if reached {
		fmt.Printf(rowFmt+"\n", goalLabel, "REACHED")
		fmt.Println(sep)
		fmt.Println()
		fmt.Println("====================================================")
		fmt.Printf(" S U C C E S S !   %s  T X / S  R E A C H E D !\n", formatNum(targetRPS))
		fmt.Println("====================================================")
	} else {
		fmt.Printf(rowFmt+"\n", goalLabel, "NOT REACHED")
		fmt.Println(sep)
		fmt.Printf("\n  Achieved %.2f%% of target (%.1fx away)\n",
			avgTPS/float64(targetRPS)*100, float64(targetRPS)/avgTPS)
	}
}

func formatFloat(f float64) string {
	if f >= 1_000_000 {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%.2f", f)
}
