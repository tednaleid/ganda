package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// Function to check if a number is prime
func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i <= int(math.Sqrt(float64(n))); i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// Function to generate prime numbers
func generatePrimes(n int) []int {
	primes := []int{}
	for i := 2; len(primes) < n; i++ {
		if isPrime(i) {
			primes = append(primes, i)
		}
	}
	return primes
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: primefit <number of primes>")
		os.Exit(1)
	}

	numPrimes, err := strconv.Atoi(os.Args[1])
	if err != nil || numPrimes < 1 {
		fmt.Println("Invalid number of primes. Please enter a positive integer.")
		os.Exit(1)
	}

	maxValue := int64(math.MaxInt64)
	primes := generatePrimes(10000) // Generate a large number of primes
	var product int64 = 1
	var i int

	for i = numPrimes; i <= len(primes); i++ {
		product = 1
		for j := i - numPrimes; j < i; j++ {
			if primes[j] > int(maxValue/product) {
				fmt.Printf("%d primes - %v = overflow - %d\n", i, primes[j-numPrimes+1:j+1], maxValue)
				return
			}
			product *= int64(primes[j])
		}
		fmt.Printf("%d primes - %v = %d : %d\n", i, primes[i-numPrimes:i], product, maxValue-product)
	}
}
