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
	"unicode"
)

//go:embed example.sh
var bashExample []byte

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

		indented := PrefixLines(bytes.NewReader(bashExample), []byte("  "))
		io.Copy(w, indented)
		w.Write([]byte{'\n'}) // Last line-prefix isn't newline-terminated.
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
// output with whitespace.
func cleanInput(input []byte) string {
	var cleaned strings.Builder
	cleaned.Grow(len(input))
	for _, char := range string(input) {
		if !unicode.IsSpace(char) {
			cleaned.WriteRune(char)
		}
	}
	return cleaned.String()
}

// PrefixLines from r with prefix.
func PrefixLines(r io.Reader, prefix []byte) io.Reader {
	return &prefixReader{
		prefix:   prefix,
		buffered: prefix,
		inner:    r,
	}
}

// prefixReader emits bytes read from inner, but also emits prefix at the start
// of every line. Use the [PrefixLines] constructor. Wrapping [io.Reader] rather
// than [io.Writer] was inspired by [prefixer].
//
// This is vanity: used to indent example code in the help docs.
//
// [prefixer]: https://github.com/goware/prefixer
type prefixReader struct {
	prefix   []byte
	buffered []byte
	inner    io.Reader
}

// Read implements io.Reader.
func (r *prefixReader) Read(p []byte) (n int, err error) {
	inter := make([]byte, 1)
	var c byte

	for i := range p {
		if len(r.buffered) != 0 {
			c, r.buffered = r.buffered[0], r.buffered[1:]
		} else {
			if _, err = r.inner.Read(inter); err != nil {
				return
			}
			c = inter[0]
			if c == byte('\n') {
				r.buffered = r.prefix[:]
			}
		}
		p[i] = c
		n++
	}

	return
}
