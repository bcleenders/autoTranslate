package main

import "bufio"
import "compress/bzip2"
import "encoding/json"
import "flag"
import "io"
import "log"
import "os"
import "runtime"

// Whether or not the files are zipped with bzip2
// Reading and parsing compressed data gives a ~30% overhead compared to uncompressed data
var isZipped bool

// Number of simultanious readers
var numReaders int
var dataRoot string

func main() {
	flag.StringVar(&dataRoot, "data", "", "The ")
	flag.IntVar(&numReaders, "readers", runtime.NumCPU(), "number of simultanious readers (default is NumCPU)")
	flag.BoolVar(&isZipped, "zipped", true, "whether the files are zipped (with bzip2)")

	flag.Parse()

	if dataRoot == "" {
		log.Println("No data folder given: I need data!")
		log.Println("Use the -data flag to provide the absolute path to the files")
		os.Exit(1)
	}

	existingRoot, _ := exists(dataRoot)
	if ! existingRoot {
		log.Println("Could not read", dataRoot)
		log.Println("Please make sure it exists and is readable")
		os.Exit(2)
	}

	// Print the current settings
	log.Println("Reading from:", dataRoot)
	log.Println("Number of readers:", numReaders)
	if isZipped {
		log.Println("Reading zipped files")
	} else {
		log.Println("Reading uncompressed files")
	}

	// Use the settings!
	runtime.GOMAXPROCS(runtime.NumCPU())

	files := []string{
		"2007/RC_2007-10",
		"2007/RC_2007-11",
		"2007/RC_2007-12",

		"2008/RC_2008-01",
		"2008/RC_2008-02",
		"2008/RC_2008-03",
		"2008/RC_2008-04",
		"2008/RC_2008-05",
		"2008/RC_2008-06",
		"2008/RC_2008-07",
		"2008/RC_2008-08",
		"2008/RC_2008-09",
		"2008/RC_2008-10",
		"2008/RC_2008-11",
		"2008/RC_2008-12",

		"2009/RC_2009-01",
		"2009/RC_2009-02",
		"2009/RC_2009-03",
		"2009/RC_2009-04",
		"2009/RC_2009-05",
		"2009/RC_2009-06",
		"2009/RC_2009-07",
		"2009/RC_2009-08",
		"2009/RC_2009-09",
		"2009/RC_2009-10",
		"2009/RC_2009-11",
		"2009/RC_2009-12",

		"2010/RC_2010-01",
		"2010/RC_2010-02",
		"2010/RC_2010-03",
		"2010/RC_2010-04",
		"2010/RC_2010-05",
		"2010/RC_2010-06",
		"2010/RC_2010-07",
		"2010/RC_2010-08",
		"2010/RC_2010-09",
		"2010/RC_2010-10",
		"2010/RC_2010-11",
		"2010/RC_2010-12",

		"2011/RC_2011-01",
		"2011/RC_2011-02",
		"2011/RC_2011-03",
		"2011/RC_2011-04",
		"2011/RC_2011-05",
		"2011/RC_2011-06",
		"2011/RC_2011-07",
		"2011/RC_2011-08",
		"2011/RC_2011-09",
		"2011/RC_2011-10",
		"2011/RC_2011-11",
		"2011/RC_2011-12",
	}

	for k, v := range files {
		files[k] = dataRoot + "/" + v
		if isZipped {
			files[k] = files[k] + ".bz2"
		}
	}

	// Let's start reading/decompressing/parsing/...
	process(files, numReaders)
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return true, err
}

var countTotal = make(chan int64)
var countErrors = make(chan int64)

// Distributes all files to the readers.
// Once all readers finished, it prints the results
// Blocks untill readers are finished and results are printed
func process(files []string, numReaders int) {
	// Init a concurrency limiter
	blocker := make(chan int, numReaders)
	for i := 0; i < numReaders; i++ {
		log.Println("Starting reader", i)
		blocker <- 1
	}

	// Init three more chans to communicate with the routines
	finished := make(chan chan int)

	go keepScore("Total", countTotal, finished)
	go keepScore("Errors", countErrors, finished)

	// Loop over the files in reverse order
	// We start with the biggest files
	// Should give a more equal finishing time
	for i := len(files)-1; i >= 0; i-- {
		// This blocks
		<-blocker

		go readFile(files[i], blocker)
	}

	// Block until everything finished.
	for i := 0; i < numReaders; i++ {
		<-blocker
		log.Println("Terminated reader", i, "-> no more files to read")
	}

	for i := 0; i < 2; i++ {
		w := make(chan int)
		finished <- w

		// Wait for it to print, then continue
		<-w
	}
}

// Function for aggregating scores between goroutines
// At the end of a job, routines can send a score. All scores will be summed.
// Prints the description and the score when it receives input on the finished channel.
func keepScore(description string, scores <-chan int64, finished <-chan chan int) {
	count := int64(0)

	for {
        // Either we wait untill we have a new URL incoming, or we quit
        select {
        case c := <-finished:
        	log.Println(description, "->", count)
        	c <- 1
        	return
        case score := <-scores:
            count += score
        }
    }
}

func readFile(file string, finished chan<- int) {
	// Don't forget to let the others know we finished here
	defer func() {
        finished <- 1
    }()

	fileReader, err := os.Open(file)
	defer fileReader.Close()
	if err != nil {
		log.Println("Error reading file", file, ":", err)
		return
	}

	// If we're handling zipped data, add a bzip2 decompressor in between
	var reader io.Reader
	if isZipped {
		reader = bzip2.NewReader(fileReader)
	} else {
		reader = fileReader
	}

	// No error -> continue!
	log.Println("Reading file", file)

	// Scan file contents
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	numLines := int64(0)
	subTotErrors := int64(0)

	// The struct we're gonna read our data into every time
	var entry = &Entry{}

	for scanner.Scan() {
		numLines++

		line := scanner.Bytes()

		err := json.Unmarshal(line, &entry)

		if err != nil {
	    	log.Println(err)
	    	log.Println(line)
	    	log.Println("\n")

	    	subTotErrors++
	    }

	    // Do something with the &entry here...
	    if entry.Author == "[deleted]" {
	    	// ... Nothing to do atm
	    }
	}

	countTotal <- numLines
	countErrors <- subTotErrors
}

type Entry struct {
	Archived			bool	`json:"archived"`
	Author				string	`json:"author"`
	Author_flair_css	string	`json:"author_flair_css"`
	Author_flair_text	string	`json:"author_flair_text"`
	Body				string	`json:"body"`
	Controversiality	int		`json:"controversiality"`
	Created_utc			string	`json:"created_utc"`
	Distinguished		string	`json:"distinguished"`
	Downs				int		`json:"downs"`
	// // Edited				int	`json:"edited"` // Boolean or timestamp -> difficult to parse
	Gilded				int		`json:"gilded"`
	Id					string	`json:"id"`
	Link_id				string	`json:"link_id"`
	Name				string	`json:"name"`
	Parent_id			string	`json:"parent_id"`
	Retrieved_on		int		`json:"retrieved_on"`
	Score				int		`json:"score"`
	Score_hidden		bool	`json:"score_hidden"`
	Subreddit			string	`json:"subreddit"`
	Subreddit_id		string	`json:"subreddit_id"`
	Ups					int		`json:"ups"`
}
