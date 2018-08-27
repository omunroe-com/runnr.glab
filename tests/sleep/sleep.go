package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [duration]", os.Args[0])
		os.Exit(1)
	}

	duration, err := time.ParseDuration(os.Args[1])
	if err != nil {
		panic(fmt.Sprintf("Couldn't parse duration argument: %v", err))
	}

	fmt.Printf("Sleeping for %s\n", duration)
	time.Sleep(duration)

	fmt.Println("Done! See you next time!")
}
