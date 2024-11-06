package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"telegram/config"
)

func GetLogFile() (*os.File, error) {

	if err := os.MkdirAll(config.Telegram.LogDirectory, 0755); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(config.Telegram.LogDirectory)
	if err != nil {
		return nil, err
	}

	var latestFile os.FileInfo
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "log_") && (latestFile == nil || file.ModTime().After(latestFile.ModTime())) {
			latestFile = file
		}
	}

	var logFile *os.File
	if latestFile != nil && latestFile.Size() < config.Telegram.MaxLogSize {
		logFile, err = os.OpenFile(filepath.Join(config.Telegram.LogDirectory, latestFile.Name()), os.O_APPEND|os.O_WRONLY, 0666)
	} else {
		newIndex := 1
		if latestFile != nil {
			parts := strings.Split(strings.TrimPrefix(latestFile.Name(), "log_"), ".")
			if len(parts) > 0 {
				lastIndex, convErr := strconv.Atoi(parts[0])
				if convErr == nil {
					newIndex = lastIndex + 1
				}
			}
		}
		logFileName := filepath.Join(config.Telegram.LogDirectory, "log_"+strconv.Itoa(newIndex)+".log")
		logFile, err = os.Create(logFileName)
	}

	return logFile, err
}
