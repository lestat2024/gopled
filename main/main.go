package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"robotlife.ai/gopled"
)

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func main() {


	if len(os.Args) < 3 {
		fmt.Println("Usage: gopled-cmd <tileSize> <stringLength>")
		return
	}

	// Convert tile size from string to int
	tileSize, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Error converting tileSize to int:", err)
		return
	}

	// Convert test string length from string to int
	stringLength, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Error converting stringLength to int:", err)
		return
	}

	// Generate two random strings of at least 4000 characters each

	slen := stringLength
	fmt.Println(slen)
	str1 := generateRandomString(slen)
	str2 := generateRandomString(slen)

	//str1 := "kitten"
	//str2 := "sitting"

	//fmt.Println("x = : ", str1)
	//fmt.Println("y = : ", str2)

	fmt.Println("Now begin ...")

	// Measure the execution time of the parallel edit distance function
	start := time.Now()

	distance := gopled.EditDistanceParallel(str1, str2, tileSize)
	duration := time.Since(start)

	fmt.Printf("Edit Distance (Parallel, Tiled & Diagonal): %d\n", distance)
	fmt.Printf("Execution Time: %s\n", duration)

	fmt.Printf("\n")
	seqstart := time.Now()
	seqdistance := gopled.EditDistance(str1, str2)
	seqduration := time.Since(seqstart)
	fmt.Printf("Edit Distance (seq): %d\n", seqdistance)
	fmt.Printf("Seq Execution Time: %s\n", seqduration)
}
