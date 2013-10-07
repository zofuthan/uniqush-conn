package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/uniqush/uniqush-conn/rpc"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func readInstanceList(r io.Reader) (list []string, err error) {
	scanner := bufio.NewScanner(r)
	list = make([]string, 0, 30)

	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	err = scanner.Err()
	return
}

var flagInputFile = flag.String("f", "", "input file (stdin by default)")
var flagTimeout = flag.Duration("timeout", 3*time.Second, "timeout")

func main() {
	flag.Parse()
	var r io.ReadCloser
	r = os.Stdin
	if *flagInputFile != "" {
		var err error
		r, err = os.Open(*flagInputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
	}

	list, err := readInstanceList(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	req := &rpc.UniqushConnInstance{}

	for _, target := range list {
		for _, peer := range list {
			req.Addr = peer
			req.Timeout = *flagTimeout
			data, err := json.Marshal(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			resp, err := http.Post(target, "application/json", bytes.NewReader(data))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}

			fmt.Printf("%v\n", string(body))
		}
	}
}