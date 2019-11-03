/*
 * b64
 *
 * a simple base64 encode/decoder for captured DNS requests
 */

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

const FORMATTER_ENCODE = 0x1
const FORMATTER_DECODE = 0x2

func main() {
	// initialize slice to hold converted payload
	var convertedPayload []byte
	var encoderMode int

	// initialize CLI parser
	inputFile := flag.String("infile", "", "input file name (required)")
	outputFile := flag.String("outfile", "", "output file name (required)")
	flag.Parse()

	// check mutually exclusive flags, and enforce one of them to be present
	if *inputFile == "" || *outputFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// bail out if input file does not exist
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Printf("ERROR: input file '%s' does not exist\n", *inputFile)
		os.Exit(1)
	}

	// read the input file
	inputPayload, readError := ioutil.ReadFile(*inputFile)
	if readError != nil {
		fmt.Printf("Error reading input file '%s': %s\n", *outputFile, readError)
		os.Exit(1)
	}

	// check input format
	_, base64DecodeErr := base64.RawURLEncoding.DecodeString(string(inputPayload))
	if base64DecodeErr != nil {
		fmt.Printf("input file '%s': looks like binary format\n", *inputFile)
		encoderMode = FORMATTER_ENCODE
	} else {
		fmt.Printf("input file '%s': looks like base64 format\n", *inputFile)
		encoderMode = FORMATTER_DECODE
	}

	// ENCODER mode: convert from BINARY to BASE64
	if encoderMode == FORMATTER_ENCODE {
		fmt.Printf("Converting to Base64\n")

		_convertedPayload := base64.RawURLEncoding.EncodeToString(inputPayload)

		// Base64 encoder delivers a string, convert it to a byte slice
		convertedPayload = []byte(_convertedPayload)
	}

	// DECODER mode: convert from BASE64 to BINARY
	if encoderMode == FORMATTER_DECODE {
		fmt.Printf("Converting to Binary\n")
		_convertedPayload, _ := base64.RawURLEncoding.DecodeString(string(inputPayload))
		convertedPayload = _convertedPayload
	}

	// write converted payload to output file
	writeError := ioutil.WriteFile(*outputFile, convertedPayload, 0644)
	if writeError != nil {
		fmt.Printf("Error reading input file '%s': %s\n", *outputFile, readError)
		os.Exit(1)
	}

	fmt.Printf("Conversion done, good bye\n")
	os.Exit(0)
}
