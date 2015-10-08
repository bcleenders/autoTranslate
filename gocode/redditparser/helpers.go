package main

import "compress/bzip2"
import "errors"
import "io"
import "log"
import "os"

func mkdir(path string) error {
	return os.MkdirAll(path, 0777)
}

// exists returns whether the given file or directory exists or not
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func getReader(file, unzippedRoot, zippedRoot string) (io.Reader, *os.File, error) {
	var readingZipped bool
	var filePath string

	if unzippedRoot != "" && exists(unzippedRoot+file) {
		filePath = unzippedRoot + file
		readingZipped = false
	} else if zippedRoot != "" && exists(zippedRoot+file+".bz2") {
		filePath = zippedRoot + file + ".bz2"
		readingZipped = true
	} else {
		log.Println("Could not read", file)
		log.Println("Tried", (unzippedRoot + file))
		log.Println("Tried", (zippedRoot + file + ".bz2"))

		return nil, nil, errors.New("Could not find file")
	}

	fileReader, err := os.Open(filePath)

	if err != nil {
		log.Println("Error reading file", file, ":", err)
		return nil, nil, err
	}

	// If we're handling zipped data, add a bzip2 decompressor in between
	if readingZipped {
		return bzip2.NewReader(fileReader), fileReader, nil
	} else {
		return fileReader, fileReader, nil
	}
}
