package main

import "encoding/json"
import "log"
import "os"
import "bufio"
import "runtime"

// const root string = "/Users/bramleenders/Downloads/reddit_data/";
const root string = "/Volumes/Gollum/reddit_data/"

var countTotal = make(chan int64)
var countDeletedAuthors = make(chan int64)
var countErrors = make(chan int64)

func main() {
	var NUM_READERS = runtime.NumCPU()
	log.Println("Number of readers:", NUM_READERS)

	runtime.GOMAXPROCS(runtime.NumCPU() + 1)

	// The hashmap we're gonna store our stuff in:
	var m map[string]int64
	m = make(map[string]int64)


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
	}

	// Init a concurrenty limiter
	blocker := make(chan int, NUM_READERS)
	for i := 0; i < NUM_READERS; i++ {
		blocker <- 1
	}

	// Init three more chans to communicate with the routines
	finished := make(chan chan int)

	go keepScore("Total", countTotal, finished)
	go keepScore("Posts with deleted authors", countDeletedAuthors, finished)
	go keepScore("Errors", countErrors, finished)

	for _, file := range files {
		// This blocks
		<-blocker

		go readFile(&m, root + file, blocker)
	}

	// Block until everything finished.
	for i := 0; i < NUM_READERS; i++ {
		<-blocker
	}

	for i := 0; i < 3; i++ {
		w := make(chan int)
		finished <- w

		// Wait for it to print, then continue
		<-w
	}
}

func keepScore(description string, scores <-chan int64, finished <-chan chan int) {
	count := int64(0)

	for {
        // Either we wait untill we have a new URL incoming, or we quit
        select {
        case c := <-finished:
        	log.Println(description, "->", count)
        	c <- 1
        case score := <-scores:
            count += score
        }
    }
}

func readFile(m *map[string]int64, file string, finished chan<- int) {
	// Don't forget to let the others know we finished here
	defer func() {
        finished <- 1
    }()

	inFile, err := os.Open(file)
	defer inFile.Close()

	if err != nil {
		log.Println("Error reading file", file, ":", err)
		return
	}
	// No error -> continue!
	log.Println("Reading file", file)

	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	subtotal := 0
	subTotDeleted := 0
	subTotErrors := 0

	for scanner.Scan() {
		subtotal++
		// log.Println(scanner.Text())
		authorDeleted, err := parse(m, scanner.Text())

		if err != nil {
			subTotErrors++
		} else if authorDeleted {
			subTotDeleted++
		}
	}

	countTotal <- int64(subtotal)
	countDeletedAuthors <- int64(subTotDeleted)
	countErrors <- int64(subTotErrors)
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
	// // Edited				int	`json:"edited"` // Boolean or timestamp
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


func parse(m *map[string]int64, line string) (bool, error) {
	var entry = &Entry{}
    err := json.Unmarshal([]byte(line), &entry)

    if err != nil {
    	log.Println(err)
    	log.Println(line)
    	log.Println(entry)
    	// log.Println("Author:", entry.Author)

    	log.Println("\n\n")
    	return false, err
    }

    if entry.Author == "[deleted]" {
    	return true, err
    } else {
    	return false, err
    }
}
