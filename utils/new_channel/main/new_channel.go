package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

func readDirectories(directories []os.FileInfo, pastChannels map[string]bool) []string {
	var actualChannels []string
	for _, f := range directories {
		if f.IsDir() {
			if !pastChannels[f.Name()] {
				fmt.Println(f.Name() + " still hasn't been created!")
				actualChannels = append(actualChannels, f.Name())
			}
		}
	}
	return actualChannels
}

func readChannelList() map[string]bool {
	var myChannels map[string]bool = make(map[string]bool)
	channelsList, err := ioutil.ReadFile("channels_list.txt")
	if err != nil {
		fmt.Println(err)
	}
	var channel string = ""
	for _, byteInFile := range string(channelsList) {
		if string(byteInFile) == "\n" {
			myChannels[channel] = true
			channel = ""
		} else {
			channel += string(byteInFile)
		}
	}
	myChannels[channel] = true // adding the last one from the file to our map
	return myChannels
}

func getDirectories() []string {
	newFile, err := os.OpenFile("channels_list.txt", os.O_RDWR, 0644)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer newFile.Close()
	directoriesPath := "../../../handlers"
	actualDirectories, err := ioutil.ReadDir(directoriesPath)
	if err != nil {
		log.Fatal(err)
	}
	pastChannels := readChannelList()
	return readDirectories(actualDirectories, pastChannels)
}

func saveNewChannels(newChannels []string) {
	channelsList, err := ioutil.ReadFile("channels_list.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	fileString := ""
	for _, ch := range string(channelsList) {
		fileString += string(ch)
	}
	var changes string = ""
	for i := range newChannels {
		changes += "\n" + newChannels[i]
	}
	b := []byte(fileString + changes)
	err = ioutil.WriteFile("channels_list.txt", b, 0644)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("New channels list updated with success!")
}

func main() {
	channelsList := getDirectories()
	saveNewChannels(channelsList)
	const filePath = "../../../cmd/courier/main.go"
	for i := 0; i < len(channelsList); i++ {
		channelPath := "github.com/nyaruka/courier/handlers/" + channelsList[i]
		index, err := getLineIndex(filePath)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Error while trying to read file!")
			return
		}
		index = index - 1
		fmt.Println(channelPath)
		err = insertStringToFile(filePath, "\t_ "+`"`+channelPath+`"`+"\n", index)
		if err != nil {
			fmt.Println("Some error occurred, please, try again!")
		}
		fmt.Println(channelsList[i])
	}
}
