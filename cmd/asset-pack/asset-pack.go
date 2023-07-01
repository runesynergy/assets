package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	"image/png"
	_ "image/png"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"azul3d.org/engine/binpack"
	"golang.org/x/exp/slices"
)

var pack_margin = 4
var visited = make(map[string]bool)
var recursive = false
var output_path string
var atlas = make(map[string]*input)
var joins = make(map[string]map[string]*input)

type ninepatch struct {
	Top    int  `json:"top"`
	Left   int  `json:"left"`
	Right  int  `json:"right"`
	Bottom int  `json:"bottom"`
	Border bool `json:"border"`
}

type input struct {
	X         int        `json:"x"`
	Y         int        `json:"y"`
	W         int        `json:"w"`
	H         int        `json:"h"`
	Ninepatch *ninepatch `json:"ninepatch,omitempty"`
	filepath  string
	img       image.Image
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (i *input) size() int {
	return max(i.W, i.H)
}

type inputs []*input

func (i inputs) Len() int {
	return len(i)
}

func (i inputs) Size(n int) (w, h int) {
	v := i[n]
	w = v.W + (pack_margin * 2)
	h = v.H + (pack_margin * 2)
	return
}

func (i inputs) Place(n, x, y int) {
	v := i[n]
	v.X = x + pack_margin
	v.Y = y + pack_margin
}

func path_no_ext(path string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)]
}

func try_file(file_path string) (result *input, err error) {
	if file_path == output_path {
		return
	}
	if visited[file_path] {
		return
	}
	visited[file_path] = true
	var data []byte
	if data, err = os.ReadFile(file_path); err != nil {
		return
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png", "image/gif":
		var img image.Image
		if img, _, err = image.Decode(bytes.NewReader(data)); err != nil {
			return
		}
		size := img.Bounds().Max
		result = &input{
			filepath: file_path,
			img:      img,
			W:        size.X,
			H:        size.Y,
		}
	}

	if result != nil {
		if data, err = os.ReadFile(file_path + ".json"); !errors.Is(err, fs.ErrNotExist) {
			var join map[string]*input
			if err = json.Unmarshal(data, &join); err != nil {
				err = fmt.Errorf("%w: unmarshalling image manifest", err)
				return
			}
			joins[file_path] = join
		} else {
			err = nil
		}
	}

	return
}

func try_dir(dir_path string) (results []*input, err error) {
	if visited[dir_path] {
		return
	}
	visited[dir_path] = true
	err = filepath.WalkDir(dir_path, func(path string, d os.DirEntry, err error) error {
		// report directory read error
		if err != nil {
			return err
		}
		if path == dir_path {
			return nil
		}
		if recursive && d.IsDir() {
			var tmp []*input
			if tmp, err = try_dir(path); err != nil {
				return err
			} else if tmp != nil {
				results = append(results, tmp...)
			}
			return nil
		}
		var input *input
		if input, err = try_file(path); err != nil {
			return err
		} else if input != nil {
			results = append(results, input)
		}
		return nil
	})
	return
}

func main() {
	flag.StringVar(&output_path, "o", "out.png", "the filepath of the output image")
	flag.IntVar(&pack_margin, "margin", 4, "sets the space between images")
	flag.BoolVar(&recursive, "recursive", false, "whether to traverse the input filepaths recursively")
	flag.Parse()

	var errors []error
	var inputs inputs

	for _, path := range flag.Args() {
		info, err := os.Stat(path)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if info.IsDir() {
			var results []*input
			if results, err = try_dir(path); err != nil {
				errors = append(errors, err)
				continue
			}
			inputs = append(inputs, results...)
		} else {
			var result *input
			if result, err = try_file(path); err != nil {
				errors = append(errors, err)
				continue
			}
			if result != nil {
				inputs = append(inputs, result)
			}
		}
	}

	slices.SortFunc(inputs, func(a, b *input) bool {
		return a.size() > b.size()
	})

	w, h := binpack.Pack(inputs)

	if w == -1 {
		for _, input := range inputs {
			fmt.Println(input)
		}
		panic("failed to pack")
	}

	if output_file, err := os.OpenFile(output_path, os.O_CREATE|os.O_WRONLY, 0655); err != nil {
		errors = append(errors, err)
	} else {
		defer output_file.Close()

		output := image.NewRGBA(image.Rect(0, 0, w, h))

		for _, input := range inputs {
			input.filepath = path_no_ext(input.filepath)
			input.filepath = strings.ReplaceAll(input.filepath, "\\", ",")
			input.filepath = strings.ReplaceAll(input.filepath, "/", ",")
			fmt.Println(input.filepath)
			for y := 0; y < input.H; y++ {
				dstY := y + input.Y
				for x := 0; x < input.W; x++ {
					dstX := x + input.X
					output.Set(dstX, dstY, input.img.At(x, y))
				}
			}
		}

		if err := png.Encode(output_file, output); err != nil {
			panic(fmt.Errorf("%w: %q", err, output_path))
		}
	}

	// export json in the form of a mapping to simplify loading into client
	for _, input := range inputs {
		atlas[input.filepath] = input
	}

	for path, join := range joins {
		path = path_no_ext(path)
		path = strings.ReplaceAll(path, "\\", ",")
		path = strings.ReplaceAll(path, "/", ",")
		base := atlas[path]
		for name, value := range join {
			value.X += base.X
			value.Y += base.Y
			atlas[name] = value
		}
	}

	manifest, err := json.Marshal(atlas)
	if err != nil {
		errors = append(errors, err)
	} else if err = os.WriteFile(output_path+".json", manifest, 0655); err != nil {
		errors = append(errors, err)
	}

	for _, err := range errors {
		fmt.Println(err)
	}
}
