package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"facette.io/natsort"
)

var (
	input_path      string
	output_path     string
	include_pattern string
	exclude_pattern string
)

type result struct {
	CRC32 uint32 `json:"crc"`
	Size  int64  `json:"size"`
}

func main() {
	flag.StringVar(&input_path, "i", ".", "The input path")
	flag.StringVar(&output_path, "o", "manifest.json", "The path for the manifest file.")
	flag.StringVar(&include_pattern, "p", ".*", "The regex pattern used for including file paths.")
	flag.StringVar(&exclude_pattern, "e", "^$", "The regex pattern used for excluding file paths.")
	flag.Parse()

	inc := regexp.MustCompile(include_pattern)
	exc := regexp.MustCompile(exclude_pattern)

	results := make(map[string]result)

	var mtx sync.Mutex
	var wg sync.WaitGroup

	filepath.Walk(input_path, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if inc.MatchString(path) && !exc.MatchString(path) {
			path := path
			size := info.Size()
			wg.Add(1)
			go func() {
				defer wg.Done()
				crc := crc32.NewIEEE()
				file, err := os.Open(path)
				if err != nil {
					fmt.Println("Failed to open file:", path)
				}
				defer file.Close()

				io.Copy(crc, file)
				crc32 := crc.Sum32()

				mtx.Lock()
				defer mtx.Unlock()

				path = strings.ReplaceAll(path, string(os.PathSeparator), "/")

				results[path] = result{
					Size:  size,
					CRC32: crc32,
				}
			}()
		}
		return err
	})

	wg.Wait()

	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	natsort.Sort(keys)

	raw, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(raw))
}
