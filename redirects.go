package main

import (
	"fmt"
	"os"
	"strings"
)

// RedirectInfo содержит информацию о редиректах
type RedirectInfo struct {
	inputFile  string
	outputFile string
	append     bool
}

// parseRedirects разбирает редиректы в команде
func parseRedirects(cmdStr string) (string, *RedirectInfo) {
	redirects := &RedirectInfo{}
	
	// Разбор редиректа ввода <
	if strings.Contains(cmdStr, "<") {
		parts := strings.Split(cmdStr, "<")
		if len(parts) == 2 {
			cmdStr = strings.TrimSpace(parts[0])
			redirects.inputFile = strings.TrimSpace(parts[1])
		}
	}
	
	// Разбор редиректа вывода > и >>
	if strings.Contains(cmdStr, ">") {
		parts := strings.Split(cmdStr, ">")
		if len(parts) == 2 {
			cmdStr = strings.TrimSpace(parts[0])
			outputFile := strings.TrimSpace(parts[1])
			
			// Проверка на append (>>)
			if strings.HasPrefix(outputFile, ">") {
				redirects.append = true
				outputFile = strings.TrimSpace(outputFile[1:])
			}
			redirects.outputFile = outputFile
		}
	}
	
	return cmdStr, redirects
}

// applyRedirects применяет редиректы к команде
func (s *Shell) applyRedirects(redirects *RedirectInfo) (*os.File, *os.File, error) {
	var stdin, stdout *os.File
	var err error
	
	// Применение редиректа ввода
	if redirects.inputFile != "" {
		stdin, err = os.Open(redirects.inputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot open input file %s: %v", redirects.inputFile, err)
		}
	}
	
	// Применение редиректа вывода
	if redirects.outputFile != "" {
		var flag int
		if redirects.append {
			flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		} else {
			flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		
		stdout, err = os.OpenFile(redirects.outputFile, flag, 0644)
		if err != nil {
			if stdin != nil {
				stdin.Close()
			}
			return nil, nil, fmt.Errorf("cannot open output file %s: %v", redirects.outputFile, err)
		}
	}
	
	return stdin, stdout, nil
}

// cleanupRedirects закрывает файлы редиректов
func cleanupRedirects(stdin, stdout *os.File) {
	if stdin != nil && stdin != os.Stdin {
		stdin.Close()
	}
	if stdout != nil && stdout != os.Stdout {
		stdout.Close()
	}
}
