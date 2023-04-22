package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	"image/png"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"

	"azul3d.org/engine/binpack"
	"golang.org/x/exp/slices"
)

var packMargin = 4
var visited = make(map[string]bool)
var recursive = false

type input struct {
	X        int `json:"x"`
	Y        int `json:"y"`
	W        int `json:"w"`
	H        int `json:"h"`
	filepath string
	img      image.Image
}

func (i *input) size() int {
	if i.W > i.H {
		return i.W
	}
	return i.H
}

type inputs []*input

func (i inputs) Len() int {
	return len(i)
}

func (i inputs) Size(n int) (w, h int) {
	v := i[n]
	w = v.W + (packMargin * 2)
	h = v.H + (packMargin * 2)
	return
}

func (i inputs) Place(n, x, y int) {
	v := i[n]
	v.X = x + packMargin
	v.Y = y + packMargin
}

func pathNoExt(path string) string {
	path = filepath.Base(path)
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)]
}

func tryFile(filepath string) (result *input, err error) {
	if visited[filepath] {
		return
	}
	visited[filepath] = true
	var data []byte
	if data, err = os.ReadFile(filepath); err != nil {
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
			filepath: filepath,
			img:      img,
			W:        size.X,
			H:        size.Y,
		}
	}
	return
}

func tryDir(dirpath string) (results []*input, err error) {
	if visited[dirpath] {
		return
	}
	visited[dirpath] = true
	err = filepath.WalkDir(dirpath, func(path string, d os.DirEntry, err error) error {
		// report directory read error
		if err != nil {
			return err
		}
		if path == dirpath {
			return nil
		}
		if recursive && d.IsDir() {
			var tmp []*input
			if tmp, err = tryDir(path); err != nil {
				return err
			} else if tmp != nil {
				results = append(results, tmp...)
			}
			return nil
		}
		var input *input
		if input, err = tryFile(path); err != nil {
			return err
		} else if input != nil {
			results = append(results, input)
		}
		return nil
	})
	return
}

func main() {
	var outputPath string
	flag.StringVar(&outputPath, "o", "out.png", "the filepath of the output image")
	flag.IntVar(&packMargin, "margin", 4, "sets the space between images")
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
			if results, err = tryDir(path); err != nil {
				errors = append(errors, err)
				continue
			}
			inputs = append(inputs, results...)
		} else {
			var result *input
			if result, err = tryFile(path); err != nil {
				errors = append(errors, err)
				continue
			}
			inputs = append(inputs, result)
		}
	}

	slices.SortFunc(inputs, func(a, b *input) bool {
		dirA := filepath.Dir(a.filepath)
		dirB := filepath.Dir(b.filepath)
		// sort by directory first
		if dirA != dirB {
			return dirA < dirB
		}
		return a.size() > b.size()
	})

	w, h := binpack.Pack(inputs)

	if w == -1 {
		for _, input := range inputs {
			fmt.Println(input)
		}
		panic("failed to pack")
	}

	if outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0655); err != nil {
		errors = append(errors, err)
	} else {
		defer outputFile.Close()

		output := image.NewRGBA(image.Rect(0, 0, w, h))

		for _, input := range inputs {
			fmt.Println(input.filepath)
			input.filepath = pathNoExt(input.filepath)
			for y := 0; y < input.H; y++ {
				dstY := y + input.Y
				for x := 0; x < input.W; x++ {
					dstX := x + input.X
					output.Set(dstX, dstY, input.img.At(x, y))
				}
			}
		}

		if err := png.Encode(outputFile, output); err != nil {
			panic(fmt.Errorf("%w: %q", err, outputPath))
		}
	}

	// export json in the form of a mapping to simplify loading into client
	atlas := make(map[string]*input)
	for _, input := range inputs {
		atlas[input.filepath] = input
	}
	manifest, err := json.Marshal(atlas)
	if err != nil {
		errors = append(errors, err)
	} else if err = os.WriteFile(outputPath+".json", manifest, 0655); err != nil {
		errors = append(errors, err)
	}

	for _, err := range errors {
		fmt.Println(err)
	}
}
