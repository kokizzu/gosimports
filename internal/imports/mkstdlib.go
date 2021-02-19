// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

// mkstdlib generates the zstdlib.go file, containing the Go standard
// library API symbols. It's baked into the binary to avoid scanning
// GOPATH in the common case.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
)

func mustOpen(name string) io.Reader {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

func api(base string) string {
	return filepath.Join(runtime.GOROOT(), "api", base)
}

var sym = regexp.MustCompile(`^pkg (\S+).*?, (?:var|func|type|const) ([A-Z]\w*)`)

var unsafeSyms = map[string]bool{"Alignof": true, "ArbitraryType": true, "Offsetof": true, "Pointer": true, "Sizeof": true}

func main() {
	var buf bytes.Buffer
	outf := func(format string, args ...interface{}) {
		fmt.Fprintf(&buf, format, args...)
	}
	outf("// Code generated by mkstdlib.go. DO NOT EDIT.\n\n")
	outf("package imports\n")
	outf("var stdlib = map[string][]string{\n")
	f := io.MultiReader(
		mustOpen(api("go1.txt")),
		mustOpen(api("go1.1.txt")),
		mustOpen(api("go1.2.txt")),
		mustOpen(api("go1.3.txt")),
		mustOpen(api("go1.4.txt")),
		mustOpen(api("go1.5.txt")),
		mustOpen(api("go1.6.txt")),
		mustOpen(api("go1.7.txt")),
		mustOpen(api("go1.8.txt")),
		mustOpen(api("go1.9.txt")),
		mustOpen(api("go1.10.txt")),
		mustOpen(api("go1.11.txt")),
		mustOpen(api("go1.12.txt")),
		mustOpen(api("go1.13.txt")),
		mustOpen(api("go1.14.txt")),
		mustOpen(api("go1.15.txt")),

		// The API of the syscall/js package needs to be computed explicitly,
		// because it's not included in the GOROOT/api/go1.*.txt files at this time.
		syscallJSAPI(),
	)
	sc := bufio.NewScanner(f)

	pkgs := map[string]map[string]bool{
		"unsafe": unsafeSyms,
	}
	paths := []string{"unsafe"}

	for sc.Scan() {
		l := sc.Text()
		if m := sym.FindStringSubmatch(l); m != nil {
			path, sym := m[1], m[2]

			if _, ok := pkgs[path]; !ok {
				pkgs[path] = map[string]bool{}
				paths = append(paths, path)
			}
			pkgs[path][sym] = true
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	sort.Strings(paths)
	for _, path := range paths {
		outf("\t%q: []string{\n", path)
		pkg := pkgs[path]
		var syms []string
		for sym := range pkg {
			syms = append(syms, sym)
		}
		sort.Strings(syms)
		for _, sym := range syms {
			outf("\t\t%q,\n", sym)
		}
		outf("},\n")
	}
	outf("}\n")
	fmtbuf, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("zstdlib.go", fmtbuf, 0666)
	if err != nil {
		log.Fatal(err)
	}
}

// syscallJSAPI returns the API of the syscall/js package.
// It's computed from the contents of $(go env GOROOT)/src/syscall/js.
func syscallJSAPI() io.Reader {
	var exeSuffix string
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	cmd := exec.Command("go"+exeSuffix, "run", "cmd/api", "-contexts", "js-wasm", "syscall/js")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}
	return bytes.NewReader(out)
}
