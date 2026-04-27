package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const TargetConsumptions = 100000

type SharedBuffer struct {
	buffer     []int
	in         int
	out        int
	mutex      sync.Mutex
	emptySlots chan struct{}
	fullSlots  chan struct{}

	occupation int
	history    []int
}

func NewSharedBuffer(N int) *SharedBuffer {
	sb := &SharedBuffer{
		buffer:     make([]int, N),
		emptySlots: make(chan struct{}, N),
		fullSlots:  make(chan struct{}, N),
		// Pre-allocate history slice to avoid reallocation overhead during critical sections
		history: make([]int, 0, TargetConsumptions*2+100),
	}

	// Initialize emptySlots semaphore with N
	for i := 0; i < N; i++ {
		sb.emptySlots <- struct{}{}
	}

	sb.history = append(sb.history, 0) // initial occupation is 0
	return sb
}

func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	if n <= 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	for i := 5; i*i <= n; i += 6 {
		if n%i == 0 || n%(i+2) == 0 {
			return false
		}
	}
	return true
}

func producer(ctx context.Context, sb *SharedBuffer, wg *sync.WaitGroup) {
	defer wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		// 1. Wait(emptySlots)
		select {
		case <-ctx.Done():
			return
		case <-sb.emptySlots:
			num := rng.Intn(10000000) + 1

			// 2. Wait(mutex) - Critical Section
			sb.mutex.Lock()
			sb.buffer[sb.in] = num
			sb.in = (sb.in + 1) % len(sb.buffer)
			sb.occupation++
			sb.history = append(sb.history, sb.occupation)
			sb.mutex.Unlock() // 3. Signal(mutex)

			// 4. Signal(fullSlots)
			select {
			case sb.fullSlots <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func consumer(ctx context.Context, sb *SharedBuffer, wg *sync.WaitGroup, itemsConsumed *int32, done chan struct{}) {
	defer wg.Done()

	for {
		// 1. Wait(fullSlots)
		select {
		case <-ctx.Done():
			return
		case <-sb.fullSlots:
			// 2. Wait(mutex) - Critical Section
			sb.mutex.Lock()
			num := sb.buffer[sb.out]
			sb.out = (sb.out + 1) % len(sb.buffer)
			sb.occupation--
			sb.history = append(sb.history, sb.occupation)
			sb.mutex.Unlock() // 3. Signal(mutex)

			// 4. Signal(emptySlots)
			select {
			case sb.emptySlots <- struct{}{}:
			case <-ctx.Done():
				return
			}

			prime := isPrime(num)

			// We atomic increment the consumed count
			count := atomic.AddInt32(itemsConsumed, 1)

			if count <= 5 {
				fmt.Printf("Consumed: %d | isPrime: %v\n", num, prime)
			}

			// 5. Check Termination Condition
			if count == TargetConsumptions {
				close(done)
				return // this specific goroutine can return, others will hit ctx.Done()
			}
		}
	}
}

func runExperiment(N, Np, Nc int, firstRun bool) time.Duration {
	sb := NewSharedBuffer(N)

	var itemsConsumed int32
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	start := time.Now()

	// Start Consumers
	for i := 0; i < Nc; i++ {
		wg.Add(1)
		go consumer(ctx, sb, &wg, &itemsConsumed, done)
	}

	// Start Producers
	for i := 0; i < Np; i++ {
		wg.Add(1)
		go producer(ctx, sb, &wg)
	}

	// Wait for $10^5$ consumptions
	<-done
	elapsed := time.Since(start)

	// Gracefully shutdown all goroutines
	cancel()
	wg.Wait()

	// If it's the first run, dump the occupation history for the graphs
	if firstRun {
		dumpHistory(N, Np, Nc, sb.history)
	}

	return elapsed
}

func dumpHistory(N, Np, Nc int, history []int) {
	filename := fmt.Sprintf("occupation_files/occupation_N%d_Np%d_Nc%d.csv", N, Np, Nc)
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Failed to create %s: %v", filename, err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	writer.Write([]string{"Operation", "Occupation"})
	for i, occ := range history {
		writer.Write([]string{strconv.Itoa(i), strconv.Itoa(occ)})
	}
}

func main() {
	// The problem parameters
	N_values := []int{1, 10, 100, 1000}
	threads := []struct{ Np, Nc int }{
		{1, 1}, {1, 2}, {1, 4}, {1, 8},
		{2, 1}, {4, 1}, {8, 1},
	}

	// Prepare export file
	f, err := os.Create("execution_times.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	csvWriter := csv.NewWriter(f)
	defer csvWriter.Flush()
	csvWriter.Write([]string{"N", "Np", "Nc", "AvgTimeMs"})

	fmt.Println("Starting Producer-Consumer Benchmark...")

	if err := os.MkdirAll("occupation_files", 0755); err != nil {
		log.Fatalf("Failed to create occupation_files directory: %v", err)
	}

	for _, N := range N_values {
		for _, t := range threads {
			var totalDuration time.Duration
			fmt.Printf("Running N=%-4d Np=%d Nc=%d ... ", N, t.Np, t.Nc)

			// Calculate average of 10 runs
			for run := 0; run < 10; run++ {
				dur := runExperiment(N, t.Np, t.Nc, run == 0) // run==0 means export history
				totalDuration += dur
			}

			avgTime := totalDuration / 10
			fmt.Printf("Avg Time: %v\n", avgTime)

			csvWriter.Write([]string{
				strconv.Itoa(N),
				strconv.Itoa(t.Np),
				strconv.Itoa(t.Nc),
				fmt.Sprintf("%d", avgTime.Milliseconds()),
			})
		}
	}
	fmt.Println("\nAll experiments completed successfully. CSV files generated in current directory.")
}
