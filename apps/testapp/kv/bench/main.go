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
	"strings"
	"sync/atomic"
	"time"
)

const targetRPS = 10_000_000

type serverStats struct {
	InjectedTxs    uint64 `json:"injected_txs"`
	ExecutedTxs    uint64 `json:"executed_txs"`
	BlocksProduced uint64 `json:"blocks_produced"`
}

func main() {
	addr := flag.String("addr", "localhost:9090", "server host:port")
	duration := flag.Duration("duration", 10*time.Second, "test duration")
	workers := flag.Int("workers", 1000, "concurrent workers")
	flag.Parse()

	fmt.Printf("Stress Test Configuration\n")
	fmt.Printf("  Server:    %s\n", *addr)
	fmt.Printf("  Duration:  %s\n", *duration)
	fmt.Printf("  Workers:   %d\n", *workers)
	fmt.Printf("  Goal:      %d req/s\n\n", targetRPS)

	if err := checkServer(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Server check failed: %v\n", err)
		os.Exit(1)
	}

	before := fetchStats(*addr)

	rawReq := fmt.Appendf(nil,
		"POST /tx HTTP/1.1\r\nHost: %s\r\nContent-Type: text/plain\r\nContent-Length: 3\r\n\r\ns=v",
		*addr,
	)

	var success atomic.Uint64
	var failures atomic.Uint64

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	done := make(chan struct{}, *workers)
	for i := 0; i < *workers; i++ {
		go worker(ctx, *addr, rawReq, &success, &failures, done)
	}

	start := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastCount uint64
	var peakRPS uint64

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			cur := success.Load() + failures.Load()
			rps := cur - lastCount
			lastCount = cur
			if rps > peakRPS {
				peakRPS = rps
			}
			elapsed := time.Since(start).Truncate(time.Second)
			fmt.Printf("\r[%6s] Total: %12d | Success: %12d | Fail: %8d | RPS: %12d   ",
				elapsed, cur, success.Load(), failures.Load(), rps)
		}
	}

	elapsed := time.Since(start)

	for i := 0; i < *workers; i++ {
		<-done
	}

	after := fetchStats(*addr)

	total := success.Load() + failures.Load()
	avgRPS := float64(total) / elapsed.Seconds()

	var txsPerBlock float64
	deltaBlocks := after.BlocksProduced - before.BlocksProduced
	deltaTxs := after.ExecutedTxs - before.ExecutedTxs
	if deltaBlocks > 0 {
		txsPerBlock = float64(deltaTxs) / float64(deltaBlocks)
	}

	reached := avgRPS >= float64(targetRPS)

	fmt.Println()
	fmt.Println()
	printResults(elapsed, uint64(*workers), total, success.Load(), failures.Load(),
		avgRPS, float64(peakRPS), deltaBlocks, deltaTxs, txsPerBlock, reached)
}

func worker(ctx context.Context, addr string, rawReq []byte, success, failures *atomic.Uint64, done chan struct{}) {
	defer func() { done <- struct{}{} }()

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
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusAccepted {
				success.Add(1)
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

func printResults(elapsed time.Duration, workers, total, success, failures uint64,
	avgRPS, peakRPS float64, blocks, executedTxs uint64, txsPerBlock float64, reached bool) {

	sep := "+----------------------------------------+----------------------------------------+"
	rowFmt := "| %-38s | %-38s |"

	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "STRESS TEST RESULTS", "")
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Duration", elapsed.Truncate(time.Millisecond).String())
	fmt.Printf(rowFmt+"\n", "Workers", fmt.Sprintf("%d", workers))
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Total Requests", formatNum(total))
	fmt.Printf(rowFmt+"\n", "Successful (202)", formatNum(success))
	fmt.Printf(rowFmt+"\n", "Failed", formatNum(failures))
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Avg req/s", formatFloat(avgRPS))
	fmt.Printf(rowFmt+"\n", "Peak req/s (1s window)", formatFloat(peakRPS))
	fmt.Println(sep)
	fmt.Printf(rowFmt+"\n", "Server Blocks Produced", formatNum(blocks))
	fmt.Printf(rowFmt+"\n", "Server Txs Executed", formatNum(executedTxs))
	fmt.Printf(rowFmt+"\n", "Avg Txs per Block", fmt.Sprintf("%.2f", txsPerBlock))
	fmt.Println(sep)

	if reached {
		fmt.Printf(rowFmt+"\n", "Goal (10M req/s)", "REACHED")
		fmt.Println(sep)
		fmt.Println()
		fmt.Println("  ====================================================")
		fmt.Println("      S U C C E S S !   1 0 M  R E A C H E D !")
		fmt.Println("  ====================================================")
	} else {
		fmt.Printf(rowFmt+"\n", "Goal (10M req/s)", "NOT REACHED")
		fmt.Println(sep)
		fmt.Printf("\n  Achieved %.2f%% of target (%.1fx away)\n",
			avgRPS/float64(targetRPS)*100, float64(targetRPS)/avgRPS)
	}
}

func formatFloat(f float64) string {
	if f >= 1_000_000 {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%.2f", f)
}
