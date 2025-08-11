package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// Shell представляет собой основной интерпретатор командной строки
type Shell struct {
	reader *bufio.Reader
	env    map[string]string
}

// NewShell создает новый экземпляр shell
func NewShell() *Shell {
	shell := &Shell{
		reader: bufio.NewReader(os.Stdin),
		env:    make(map[string]string),
	}
	
	// Инициализация переменных окружения
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			shell.env[pair[0]] = pair[1]
		}
	}
	
	return shell
}

// Run запускает основной цикл shell
func (s *Shell) Run() {
	// Настройка обработки сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	fmt.Println("Welcome to GoShell! Type 'exit' to quit.")
	
	for {
		fmt.Print("gosh> ")
		
		// Чтение команды
		input, err := s.reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				os.Exit(0)
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}
		
		// Удаление символа новой строки
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		// Обработка команды exit
		if input == "exit" {
			fmt.Println("Goodbye!")
			os.Exit(0)
		}
		
		// Выполнение команды
		s.executeCommand(input)
	}
}

// executeCommand выполняет команду
func (s *Shell) executeCommand(input string) {
	// Подстановка переменных окружения
	input = s.expandEnvironmentVariables(input)
	
	// Разбор команд с условным выполнением
	commands := s.parseConditionalCommands(input)
	
	for _, cmd := range commands {
		if cmd.condition == "&&" && !cmd.shouldExecute {
			continue
		}
		if cmd.condition == "||" && cmd.shouldExecute {
			continue
		}
		
		// Разбор конвейеров
		pipelines := s.parsePipelines(cmd.command)
		if len(pipelines) == 1 {
			// Простая команда
			s.executeSimpleCommand(pipelines[0])
		} else {
			// Конвейер команд
			s.executePipeline(pipelines)
		}
		
		cmd.shouldExecute = true
	}
}

// Command представляет команду с условием выполнения
type Command struct {
	command       string
	condition     string
	shouldExecute bool
}

// parseConditionalCommands разбирает команды с && и ||
func (s *Shell) parseConditionalCommands(input string) []*Command {
	parts := strings.Split(input, "&&")
	var commands []*Command
	
	for i, part := range parts {
		if i == 0 {
			commands = append(commands, &Command{command: strings.TrimSpace(part), condition: "", shouldExecute: true})
		} else {
			commands = append(commands, &Command{command: strings.TrimSpace(part), condition: "&&", shouldExecute: true})
		}
	}
	
	// Разбор || операторов
	var finalCommands []*Command
	for _, cmd := range commands {
		orParts := strings.Split(cmd.command, "||")
		for j, orPart := range orParts {
			if j == 0 {
				finalCommands = append(finalCommands, &Command{command: strings.TrimSpace(orPart), condition: cmd.condition, shouldExecute: true})
			} else {
				finalCommands = append(finalCommands, &Command{command: strings.TrimSpace(orPart), condition: "||", shouldExecute: true})
			}
		}
	}
	
	return finalCommands
}

// parsePipelines разбирает конвейер команд
func (s *Shell) parsePipelines(input string) []string {
	return strings.Split(input, "|")
}

// executeSimpleCommand выполняет простую команду
func (s *Shell) executeSimpleCommand(cmdStr string) {
	// Разбор редиректов
	cleanCmdStr, redirects := parseRedirects(cmdStr)
	
	// Разбор команды на части
	parts := strings.Fields(cleanCmdStr)
	if len(parts) == 0 {
		return
	}
	
	command := parts[0]
	args := parts[1:]
	
	// Проверка встроенных команд
	if s.isBuiltinCommand(command) {
		s.executeBuiltinCommand(command, args)
		return
	}
	
	// Выполнение внешней команды с редиректами
	s.executeExternalCommandWithRedirects(command, args, redirects)
}

// isBuiltinCommand проверяет, является ли команда встроенной
func (s *Shell) isBuiltinCommand(command string) bool {
	builtins := []string{"cd", "pwd", "echo", "kill", "ps"}
	for _, builtin := range builtins {
		if command == builtin {
			return true
		}
	}
	return false
}

// executeBuiltinCommand выполняет встроенную команду
func (s *Shell) executeBuiltinCommand(command string, args []string) {
	switch command {
	case "cd":
		s.builtinCD(args)
	case "pwd":
		s.builtinPWD()
	case "echo":
		s.builtinEcho(args)
	case "kill":
		s.builtinKill(args)
	case "ps":
		s.builtinPS()
	}
}

// builtinCD реализует команду cd
func (s *Shell) builtinCD(args []string) {
	var path string
	if len(args) == 0 {
		path = s.env["HOME"]
	} else {
		path = args[0]
	}
	
	// Подстановка переменных окружения в путь
	path = s.expandEnvironmentVariables(path)
	
	if err := os.Chdir(path); err != nil {
		fmt.Printf("cd: %v\n", err)
	}
}

// builtinPWD реализует команду pwd
func (s *Shell) builtinPWD() {
	if pwd, err := os.Getwd(); err == nil {
		fmt.Println(pwd)
	} else {
		fmt.Printf("pwd: %v\n", err)
	}
}

// builtinEcho реализует команду echo
func (s *Shell) builtinEcho(args []string) {
	for i, arg := range args {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(arg)
	}
	fmt.Println()
}

// builtinKill реализует команду kill
func (s *Shell) builtinKill(args []string) {
	if len(args) == 0 {
		fmt.Println("kill: usage: kill <pid>")
		return
	}
	
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Printf("kill: invalid pid: %s\n", args[0])
		return
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("kill: process not found: %v\n", err)
		return
	}
	
	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("kill: failed to send signal: %v\n", err)
	}
}

// builtinPS реализует команду ps
func (s *Shell) builtinPS() {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("ps: %v\n", err)
		return
	}
	fmt.Print(string(output))
}

// executeExternalCommand выполняет внешнюю команду
func (s *Shell) executeExternalCommand(command string, args []string) {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing %s: %v\n", command, err)
	}
}

// executeExternalCommandWithRedirects выполняет внешнюю команду с редиректами
func (s *Shell) executeExternalCommandWithRedirects(command string, args []string, redirects *RedirectInfo) {
	cmd := exec.Command(command, args...)
	
	// Применение редиректов
	stdin, stdout, err := s.applyRedirects(redirects)
	if err != nil {
		fmt.Printf("Error applying redirects: %v\n", err)
		return
	}
	defer cleanupRedirects(stdin, stdout)
	
	// Настройка stdin/stdout
	if stdin != nil {
		cmd.Stdin = stdin
	} else {
		cmd.Stdin = os.Stdin
	}
	
	if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing %s: %v\n", command, err)
	}
}

// executePipeline выполняет конвейер команд
func (s *Shell) executePipeline(pipelines []string) {
	if len(pipelines) == 0 {
		return
	}
	
	// Создание каналов для связи между процессами
	var pipes []*os.File
	for i := 0; i < len(pipelines)-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			fmt.Printf("Error creating pipe: %v\n", err)
			return
		}
		pipes = append(pipes, r, w)
	}
	
	// Запуск всех команд в конвейере
	var processes []*exec.Cmd
	for i, pipeline := range pipelines {
		parts := strings.Fields(pipeline)
		if len(parts) == 0 {
			continue
		}
		
		cmd := exec.Command(parts[0], parts[1:]...)
		
		// Настройка stdin/stdout для конвейера
		if i == 0 {
			// Первая команда читает из stdin
			cmd.Stdin = os.Stdin
		} else {
			// Остальные команды читают из предыдущего канала
			cmd.Stdin = pipes[(i-1)*2]
		}
		
		if i == len(pipelines)-1 {
			// Последняя команда пишет в stdout
			cmd.Stdout = os.Stdout
		} else {
			// Остальные команды пишут в следующий канал
			cmd.Stdout = pipes[i*2+1]
		}
		
		cmd.Stderr = os.Stderr
		
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting command: %v\n", err)
			return
		}
		
		processes = append(processes, cmd)
	}
	
	// Ожидание завершения всех команд
	for _, cmd := range processes {
		cmd.Wait()
	}
	
	// Закрытие каналов
	for _, pipe := range pipes {
		pipe.Close()
	}
}

// expandEnvironmentVariables подставляет переменные окружения
func (s *Shell) expandEnvironmentVariables(input string) string {
	// Простая подстановка $VAR
	for key, value := range s.env {
		placeholder := "$" + key
		input = strings.ReplaceAll(input, placeholder, value)
	}
	return input
}
