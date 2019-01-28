package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/reconquest/karma-go"
)

var (
	reWord  = regexp.MustCompile("[^\\s]+")
	columns = map[string]int{
		"usr":    2,
		"sys":    4,
		"iowait": 5,
		"idle":   11,
	}
)

type NodeData struct {
	Body   string
	Host   string
	Stream string
}

func runStat(
	hosts []string,
	username string,
	output string,
	interval string,
) (*sync.WaitGroup, *exec.Cmd, error) {
	params := []string{
		"-y",
		"-u", username,
		"-x",
		"-s",
		"--json",
		"-C", "--",
		"mpstat", interval,
	}

	cmd := exec.Command("orgalorg", params...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	cmd.Stderr = os.Stderr

	stdin.Write([]byte(strings.Join(hosts, "\n") + "\n"))
	stdin.Close()

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			text := scanner.Text()

			err := collectStatJSON(text, output)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		err = cmd.Wait()
		if err != nil {
			log.Errorf(err, "error while waiting for orgalorg")
		}
	}()

	return wg, cmd, nil
}

func collectStatJSON(text string, output string) error {
	var data NodeData
	err := json.Unmarshal([]byte(text), &data)
	if err != nil {
		return karma.Format(
			err,
			"unable to unmarshal json: %s", text,
		)
	}

	if data.Stream != "stdout" {
		return nil
	}

	// header of mpstat
	if strings.HasPrefix(data.Body, "Linux") {
		return nil
	}

	if data.Body == "\n" {
		return nil
	}

	// last line of mpstat
	if strings.HasPrefix(data.Body, "Average") {
		return nil
	}

	// columns of mpstat
	if strings.Contains(data.Body, "%idle") {
		return nil
	}

	chunks := reWord.FindAllString(data.Body, -1)

	return writeChunks(output, data.Host, chunks)
}

func writeChunks(output string, host string, chunks []string) error {
	time := chunks[0]

	for name, index := range columns {
		value := chunks[index]

		err := writeChunk(output, host, time, name, value)
		if err != nil {
			return err
		}
	}

	idle, err := strconv.ParseFloat(chunks[columns["idle"]], 64)
	if err != nil {
		return err
	}

	err = writeChunk(output, host, time, "total", fmt.Sprint(100.0-idle))
	if err != nil {
		return err
	}

	return nil
}

func writeChunk(output string, host string, time string, name string, value string) error {
	line := time + " " + value + "\n"

	dir := filepath.Join(output, host)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	file.WriteString(line)

	err = file.Close()
	if err != nil {
		return err
	}

	//log.Infof(nil, "%v: %v %v %v", host, time, name, value)

	return nil
}
