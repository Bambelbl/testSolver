package main

import (
	"fmt"
	"github.com/Bambelbl/testSolver/internal/solver"
	"go.uber.org/zap"
	"os"
	"strconv"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Specify the number of streams by argument")
		os.Exit(1)
	}
	streamsCount, err := strconv.Atoi(os.Args[1])
	if err != nil || streamsCount < 0 {
		fmt.Println("Number of streams must be a positive integer")
		os.Exit(1)
	}

	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("LOGGER INITIALIZATION ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	defer func(zapLogger *zap.Logger) {
		err = zapLogger.Sync()
		if err != nil {

		}
	}(zapLogger)
	logger := zapLogger.Sugar()

	passedTests := 0
	wg := sync.WaitGroup{}
	for i := 1; i <= streamsCount; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, id int) {
			defer wg.Done()
			ts := solver.NewTestSolver(id, "http://147.78.65.149", logger)
			passedTests += ts.Solve()
		}(&wg, i)
	}
	wg.Wait()
	if passedTests == streamsCount {
		fmt.Println("All tests passed")
	} else {
		fmt.Println("Ooops, something went wrong")
	}
}
