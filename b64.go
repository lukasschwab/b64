package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

//go:embed example.sh
var bashExample string

func main() {
	// Define flags variables
	var (
		decodeMode bool
		encodeMode bool
		urlMode    bool
	)

	//nolint:errcheck
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprint(w, "b64 encodes and decodes base-64 strings.\n\n")
		flag.PrintDefaults()
		fmt.Fprint(w, "\nExample usage:\n\n")
		indented := withPrefix(w, []byte("  "))
		fmt.Fprint(indented, bashExample)
	}

	flag.BoolVar(&decodeMode, "d", false, "Decode the input (default behavior)")
	flag.BoolVar(&encodeMode, "e", false, "Encode the input")
	flag.BoolVar(&urlMode, "u", false, "Use URL encoding (base64url) instead of standard")

	flag.Parse()

	var enc = base64.StdEncoding
	if urlMode {
		enc = base64.URLEncoding
	}

	rawInput, err := getRawInput()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if len(rawInput) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if encodeMode {
		encoded := enc.EncodeToString(rawInput)
		fmt.Println(encoded)
		return
	}

	input := cleanInput(rawInput)
	decoded, err := enc.DecodeString(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding input: %v\n", err)
		os.Exit(1)
	}
	if _, err := fmt.Println(string(decoded)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
}

func getRawInput() ([]byte, error) {
	if flag.NArg() > 0 {
		arg := flag.Arg(0)
		// If valid, treat argument as filepath.
		if info, err := os.Stat(arg); err == nil && !info.IsDir() {
			input, err := os.ReadFile(arg)
			if err != nil {
				return nil, fmt.Errorf("error reading file '%s': %w", arg, err)
			}
			return input, nil
		}
		return []byte(arg), nil
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("error reading from stdin: %w", err)
	}
	return input, nil
}

// cleanInput: base64 decoders in Go are strict, but CLI tools often produce
// output with newlines.
func cleanInput(input []byte) string {
	cleanInput := string(bytes.TrimSpace(input))
	cleanInput = strings.ReplaceAll(cleanInput, "\n", "")
	cleanInput = strings.ReplaceAll(cleanInput, "\r", "")
	cleanInput = strings.ReplaceAll(cleanInput, "\t", "")
	cleanInput = strings.ReplaceAll(cleanInput, " ", "")
	return cleanInput
}

// prefixer is a purely vain part of this program: an io.Writer that writes its
// prefix at the start of each written line, including before the first written
// byte.
//
// Use the [withPrefix] constructor.
type prefixer struct {
	lastCharWritten byte
	prefix          []byte
	inner           io.Writer
}

var _ io.Writer = (*prefixer)(nil)

func (pre *prefixer) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if pre.lastCharWritten == '\n' {
			if _, err := pre.inner.Write(pre.prefix); err != nil {
				return n, err
			}
		}
		d, err := pre.inner.Write([]byte{b})
		n += d
		pre.lastCharWritten = b
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func withPrefix(inner io.Writer, prefix []byte) io.Writer {
	return &prefixer{
		lastCharWritten: '\n',
		prefix:          prefix,
		inner:           inner,
	}
}
