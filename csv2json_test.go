package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func Test_getArgs(t *testing.T) {
	tests := []struct {
		name    string
		want    args
		wantErr bool
		osArgs  []string
	}{
		{"Default arguments", args{[]string{"test.csv"}, false}, false, []string{"./csv2json", "test.csv"}},
		{"No arguments", args{}, true, []string{"./csv2json"}},
		{"Pretty enabled", args{[]string{"test.csv"}, true}, false, []string{"./csv2json", "--pretty", "test.csv"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// save the original os.Args reference
			actualOsArgs := os.Args
			// restore the original os.Args reference and reset flag
			defer func() {
				os.Args = actualOsArgs
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			}()

			os.Args = tt.osArgs // Setting the specific command args for this test
			got, err := getArgs()
			if (err != nil) != tt.wantErr {
				t.Errorf("getArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArgs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkIfFileValid(t *testing.T) {
	// create a temporary CSV file
	tmpFile, err := ioutil.TempFile("", "test*.csv")
	check(err)

	defer os.Remove(tmpFile.Name())

	// Defining the struct we're going to use
	tests := []struct {
		name     string
		filePath string
		want     bool
		wantErr  bool
	}{
		{"File is not csv", "test.txt", false, true},
		{"File exists", tmpFile.Name(), true, false},
		{"File doesn't exist", "nowhere/test.csv", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkIfFileValid(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkIfFileValid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkIfFileValid() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processCsvFile(t *testing.T) {
	expected := []*student{&student{
		No: 1,
		Name: "name",
		Score: 0,
		Hobby: "hobby",
	}}

	tests := []struct {
		name string
		csvString string // The content of our tested CSV file
	}{
		{"Normal case", "No,Name,Score,Hobby\n1,name,0,hobby\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create a temporary CSV file
			tmpFile, err := ioutil.TempFile("", "test*.csv")
			check(err)

			defer os.Remove(tmpFile.Name())
			_, err = tmpFile.WriteString(tt.csvString)
			tmpFile.Sync()

			channel := make(chan []*student)
			wg := new(sync.WaitGroup)
			wg.Add(1)
			// Calling the targeted function as a go routine
			go processCsvFile(tmpFile.Name(), channel, wg)

			record := <-channel // Waiting for the record that we want to compare
			if !reflect.DeepEqual(record, expected) { // Making the corresponding test assertion
				t.Errorf("processCsvFile() = %v, want %v", record, expected)
			}
		})
	}
}

func Test_writeJsonFile(t *testing.T) {
	expected := []*student{&student{
		No: 1,
		Name: "name",
		Score: 0,
		Hobby: "hobby",
	}}

	tests := []struct {
		name         string
		jsonPath     string
		wantJsonPath string
		pretty       bool
	}{
		{"Compact JSON", "students-test.json", "compact.json", false},
		{"Pretty JSON", "students-test.json", "pretty.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := make(chan []*student)
			done := make(chan bool)
			// Running a go-routine to simulate processCsvFile
			go func() {
				channel <- expected
				close(channel)
			}()

			go writeJsonFile(tt.jsonPath, channel, done, tt.pretty)
			<-done

			testOutput, err := ioutil.ReadFile(tt.jsonPath)
			if err != nil {
				t.Errorf("writeJSONFile(), Output file got error: %v", err)
			}

			defer os.Remove(tt.jsonPath)

			wantOutput, err := ioutil.ReadFile(filepath.Join("testJsonFiles", tt.wantJsonPath))
			check(err)

			if (string(testOutput)) != (string(wantOutput)) {
				t.Errorf("writeJSONFile() = %v, want %v", string(testOutput), string(wantOutput))
			}
		})
	}
}
