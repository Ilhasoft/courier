package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func fileToPath(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return linesFromReader(file)
}

func linesFromReader(reader io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func insertStringToFile(path string, str string, index int) error {
	lines, err := fileToPath(path)
	if err != nil {
		return err
	}
	fileContent := ""
	for i, line := range lines {
		if i == index {
			fileContent += str
		}
		fileContent += string(line)
		fileContent += "\n"
	}
	return ioutil.WriteFile(path, []byte(fileContent), 0644)
}

func getLineIndex(path string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, err
	}

	i := 0
	code := string(data)
	var parenthesis int = 0
	var line int = 0
	for i = 0; i < len(code); i++ {
		s := string(code[i])
		if s == "(" || s == ")" {
			parenthesis++
		}
		if parenthesis == 2 {
			break
		}
		if s == "\n" {
			line++
		}
	}
	return line - 3, nil //returning 3 positions before the last parenthesis, to assure the correct index
}

func main() {
	path := flag.String("path", "../foo/main.go", "A path to main.go")
	moduleURL := flag.String("module_url", "github.com/your/module/url", "A URL to your new channel")
	flag.Parse()
	if *path == "../foo/main.go" || *moduleURL == "github.com/your/module/url" {
		fmt.Println("Please, type the correct command line args")
		return
	}

	index, err := getLineIndex(*path)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error while trying to read file!")
		return
	}

	err = insertStringToFile(*path, *moduleURL+"\n", index)
	if err != nil {
		fmt.Println("Some error occurred, please, try again!")
	}
}
