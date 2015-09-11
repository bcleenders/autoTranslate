package main

import "bufio"
import "encoding/json"
import "flag"
import "fmt"
import "log"
import "os"
import "runtime"
import "strconv"
import "strings"

// Number of simultanious readers
var numReaders int

// Path to our data
var zippedRoot string
var unzippedRoot string

// Path to output folder
var outputRoot string

// Where we start/end
var startYear int
var lastYear int

func main() {
	flag.StringVar(&zippedRoot, "zipped", "", "The root folder. From this folder everything must follow the YEAR/RC_YEAR-MONTH structure")
	flag.StringVar(&unzippedRoot, "unzipped", "", "The root folder of unzipped data.")

	flag.StringVar(&outputRoot, "out", "", "The output folder")

	flag.IntVar(&numReaders, "readers", runtime.NumCPU(), "number of simultanious readers (default is NumCPU)")

	flag.IntVar(&startYear, "start", 2007, "first year to be processed (2007 <= start <= end <= 2015)")
	flag.IntVar(&lastYear, "last", 2007, "last year to be processed (2007 <= start <= end <= 2015)")

	flag.Parse()

	// Make sure either the zipped or unzipped data path is set
	if zippedRoot == "" && unzippedRoot == "" {
		log.Println("No data folder given: I need data!")
		log.Println("Use -zipped or -unzipped")
		os.Exit(1)
	}

	if outputRoot == "" {
		log.Println("I need an output directory. Provide with -out flag")
		os.Exit(2)
	}

	// Make sure the path to zipped files exists (if it's set)
	if zippedRoot != "" && !exists(zippedRoot) {
		log.Println("Could not read from", zippedRoot)
		log.Println("Please make sure it exists and is readable")
		os.Exit(2)
	}

	// Make sure the path to unzipped files exists (if it's set)
	if unzippedRoot != "" && !exists(unzippedRoot) {
		log.Println("Could not read from", zippedRoot)
		log.Println("Please make sure it exists and is readable")
		os.Exit(2)
	}

	// Make sure the date range is valid
	if 2007 > startYear || startYear > lastYear || lastYear > 2015 {
		log.Println("Invalid data range (from", startYear, "to", lastYear, ")")
		log.Println("Must be: 2007 <= start <= end <= 2015")
		os.Exit(3)
	}

	log.Println("Processing date range", startYear, "to", lastYear)

	// Print the current settings
	log.Println("Searching for zipped files in:", zippedRoot)
	log.Println("Searching for unzipped files in:", unzippedRoot)

	os.MkdirAll(outputRoot, 0777)
	log.Println("Writing output to:", outputRoot)

	log.Println("Number of readers:", numReaders)

	// Use the settings!
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Generate a list of filenames we're gonna read
	files := getFilePaths()

	// Let's start reading/decompressing/parsing/...
	readFiles(files, numReaders)
}

// Generates the (relative) filenames for a certain period (i.e. 2007 to 2010)
func getFilePaths() []string {
	var files []string

	for year := startYear; year <= lastYear; year++ {
		os.MkdirAll(outputRoot+"/"+strconv.Itoa(year), 0777)

		startMonth := 1
		if year == 2007 {
			startMonth = 10
		}
		endMonth := 12
		if year == 2015 {
			endMonth = 5
		}

		for month := startMonth; month <= endMonth; month++ {
			filename := fmt.Sprintf("/%d/RC_%d-%02d", year, year, month)

			files = append(files, filename)
		}
	}

	return files
}

var countTotal = make(chan int64)
var countErrors = make(chan int64)

// Distributes all files to the readers.
// Once all readers finished, it prints the results
// Blocks untill readers are finished and results are printed
func readFiles(files []string, numReaders int) {
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
	for i := len(files) - 1; i >= 0; i-- {
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

	reader, handle, err := getReader(file, unzippedRoot, zippedRoot)
	if err != nil {
		return
	}
	defer handle.Close()

	// Create an output file to write our data to. Keeps same structure as input data (/YEAR/RC_YEAR-MONTH)
	output, err := os.Create(outputRoot + "/" + file)
	if err != nil {
		log.Println("Could not create", (outputRoot + file))
		log.Println(err)
		return
	}
	defer output.Close()
	writer := bufio.NewWriter(output)

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

		// Extract whatever info we want from it
		processed := process(entry)

		writer.WriteString(processed)
		writer.WriteString("\n")
	}

	// Flush data to disk to make sure everything is saved
	writer.Flush()

	countTotal <- numLines
	countErrors <- subTotErrors
}

type Entry struct {
	// Archived			bool	`json:"archived"`
	Author string `json:"author"`
	// Author_flair_css	string	`json:"author_flair_css"`
	// Author_flair_text	string	`json:"author_flair_text"`
	Body string `json:"body"`
	// Controversiality	int		`json:"controversiality"`
	// Created_utc			string	`json:"created_utc"`
	// Distinguished		string	`json:"distinguished"`
	// Downs				int		`json:"downs"`
	// // Edited				int	`json:"edited"` // Boolean or timestamp -> difficult to parse
	// Gilded				int		`json:"gilded"`
	// Id					string	`json:"id"`
	// Link_id				string	`json:"link_id"`
	// Name				string	`json:"name"`
	// Parent_id			string	`json:"parent_id"`
	// Retrieved_on		int		`json:"retrieved_on"`
	// Score				int		`json:"score"`
	// Score_hidden		bool	`json:"score_hidden"`
	// Subreddit			string	`json:"subreddit"`
	// Subreddit_id		string	`json:"subreddit_id"`
	// Ups					int		`json:"ups"`
}

func process(entry *Entry) string {
	body := entry.Body

	// Remove newlines
	body = strings.Replace(body, "\n", "", -1)

	return body
}
