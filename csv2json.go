package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gocarina/gocsv"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type args struct {
	filePaths []string
	pretty    bool
}

type student struct {
	No    int    `csv:"No" json:"no"`
	Name  string `csv:"Name" json:"name"`
	Score int    `csv:"Score" json:"score"`
	Hobby string `csv:"Hobby" json:"hobby"`
}

func getArgs() (args, error) {
	if len(os.Args) < 2 {
		return args{}, errors.New("A filepath argument is required")
	}

	pretty := flag.Bool("pretty", false, "Generate pretty JSON")
	flag.Parse()
	filePaths := flag.Args()

	return args{filePaths, *pretty}, nil
}

func checkIfFileValid(filePath string) (bool, error) {
	// check if it's a csv file
	if fileExtension := filepath.Ext(filePath); fileExtension != ".csv" {
		return false, fmt.Errorf("file %s is not CSV", filePath)
	}
	// check if file exists
	if _, err := os.Stat(filePath); err != nil && os.IsNotExist(err) {
		return false, fmt.Errorf("file %s does not exist", filePath)
	}

	return true, nil
}

func processCsvFile(filePath string, channel chan<- []*student, wg *sync.WaitGroup) {
	file, err := os.Open(filePath)
	check(err)
	// load the file content to students slice
	var students []*student
	err = gocsv.UnmarshalFile(file, &students)
	check(err)

	channel <- students

	defer wg.Done()
	defer file.Close()
}

func check(err error) {
	if err != nil {
		_, err = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func writeJsonFile(jsonPath string, writerChannel <-chan []*student, done chan<- bool, pretty bool) {
	fmt.Println("Writing JSON file...")
	var file []byte
	var record []*student
	for {
		// Waiting for pushed records into our writerChannel
		tmpRecord, more := <-writerChannel
		if len(tmpRecord) > 0 {
			record = append(record, tmpRecord...)
		}
		// run when channel closed
		if !more {
			if pretty {
				file, _ = json.MarshalIndent(record, "", "    ")
			} else {
				file, _ = json.Marshal(record)
			}
			_ = ioutil.WriteFile(jsonPath, file, 0644)
			fmt.Println("Completed!")
			done <- true
			break
		}
	}
}

func main() {
	// Showing useful information when the user enters the --help option
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <csvFile>\nOptions:\n", "./csv2json")
		flag.PrintDefaults()
	}

	// get the arguments that was entered by the user
	args, err := getArgs()
	check(err)

	// Validating the file entered
	for _, filePath := range args.filePaths {
		_, err := checkIfFileValid(filePath)
		check(err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(args.filePaths))

	// make channel for students data from csv file and done signal
	channel := make(chan []*student)
	done := make(chan bool)

	for _, filePath := range args.filePaths {
		go processCsvFile(filePath, channel, wg)
	}
	go writeJsonFile("students.json", channel, done, args.pretty)
	wg.Wait()
	close(channel)

	// wait for done signal to close the application
	<-done
}