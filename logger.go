package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/mattn/go-runewidth"
)

const (
	ColorReset = "\033[0m"
	//ColorBlack   = "\033[30m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorPurple  = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorDefault = "\033[39m"

	ColorBackgroundBlack = "\033[40m"
	//ColorBackgroundRed     = "\033[41m"
	//ColorBackgroundGreen   = "\033[42m"
	//ColorBackgroundYellow  = "\033[43m"
	//ColorBackgroundBlue    = "\033[44m"
	//ColorBackgroundPurple  = "\033[45m"
	//ColorBackgroundCyan    = "\033[46m"
	//ColorBackgroundWhite   = "\033[47m"
	//ColorBackgroundDefault = "\033[49m"

	Bold      = "\033[1m"
	Underline = "\033[4m"
	Inverse   = "\033[7m"

	logPath = "./log"

	Fatal LogLevel = iota
	Error
	Warning
	Info
	Text
	Debug
)

type LogLevel uint8

var globalLogger = New(Debug)

type Logger interface {
	GlobalLevel(lvl LogLevel)
	ToFile(message string, args ...interface{})
	logToConsole(level LogLevel, message string, args ...interface{})
	GetText(level LogLevel, message string, args ...interface{}) string

	Fatal(message string, args ...interface{})
	Error(message string, args ...interface{})
	Warning(message string, args ...interface{})
	Info(message string, args ...interface{})
	Text(message string, args ...interface{})
	Debug(message string, args ...interface{})

	StartInline()
	StopInline()
}

type logger struct {
	file      *os.File
	mutex     sync.Mutex
	writer    io.Writer
	curr      time.Time
	globalLvl LogLevel
	inline    bool
}

func createLogFile() *os.File {
	yy, mm, dd := time.Now().Date()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		e := os.Mkdir(logPath, os.ModePerm)
		if e != nil {
			globalLogger.logToConsole(Fatal, e.Error())
			return nil
		}
	}
	file, err := os.OpenFile(fmt.Sprintf(logPath+"/%d-%d-%d.log", dd, mm, yy), os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return nil
	}
	return file
}

func New(lvl LogLevel) Logger {
	return &logger{file: nil, writer: nil, mutex: sync.Mutex{}, globalLvl: lvl}
}

// GlobalLevel set max LogLevel which can be printed to console
func (p *logger) GlobalLevel(lvl LogLevel) {
	p.globalLvl = lvl
}

func (p *logger) ToFile(message string, args ...interface{}) {
	if p.file == nil {
		p.file = createLogFile()
		if p.file == nil {
			return
		}
		p.writer = p.file
	}
	pc, _, l, _ := runtime.Caller(1)
	str := ""
	for _, arg := range args {
		str += fmt.Sprintf(" [%v]", arg)
	}

	str = fmt.Sprint(time.Now().Format("15:04:05.000000"), "\t", runtime.FuncForPC(pc).Name(), "[", l, "]", "\t", message, "\t", str, "\n")
	str = strings.Replace(str, "[\n]", "\n\t", -1)
	p.mutex.Lock()
	if time.Now().Day() != p.curr.Day() {
		f := p.file
		p.file = createLogFile()
		if p.file == nil {
			return
		}
		p.writer = p.file
		_ = f.Close()
		p.curr = time.Now()
	}
	_, _ = p.writer.Write([]byte(str))
	p.mutex.Unlock()
}

func (p *logger) logToConsole(level LogLevel, message string, args ...interface{}) {
	if p.globalLvl < level {
		return
	}
	if p.inline {
		str := strings.Builder{}
		str.WriteString("\r")
		pbwidth := getWidth()
		for runewidth.StringWidth(str.String()) < pbwidth {
			str.WriteString(" ")
		}
		fmt.Print(str.String())
		fmt.Print("\r" + p.GetText(level, message, args...))
	} else {
		fmt.Println(p.GetText(level, message, args...))
	}
}

func (p *logger) GetText(level LogLevel, message string, args ...interface{}) string {
	if p.globalLvl < level {
		return ""
	}
	pc, _, l, _ := runtime.Caller(3)

	var marker string
	var msgColor string
	var argsColor string
	timeStamp := ColorWhite + time.Now().Format("15:04:05.000000")
	finfo := fmt.Sprint(ColorWhite, runtime.FuncForPC(pc).Name(), " [", l, "]")
	switch level {
	case Fatal:
		marker = ColorReset + Bold + Inverse + ColorRed + Bold + "PAN" + ColorReset
		msgColor = ColorReset + ColorRed + ColorBackgroundBlack
		argsColor = Underline
	case Error:
		marker = ColorReset + Inverse + ColorRed + Bold + "ERR" + ColorReset
		msgColor = ColorReset + ColorRed
		argsColor = ColorReset + ColorWhite + Underline
	case Warning:
		marker = ColorReset + Inverse + ColorYellow + "WRN" + ColorReset
		msgColor = Bold
		argsColor = ColorReset + ColorWhite
	case Info:
		marker = ColorReset + Inverse + ColorBlue + "INF" + ColorReset
		msgColor = ColorReset + ColorCyan
		argsColor = ColorReset + ColorBlue
		finfo = ""
	case Debug:
		marker = ColorReset + Inverse + ColorPurple + "DBG" + ColorReset
		msgColor = ColorReset + ColorGreen
		argsColor = ColorReset + ColorBlue
	case Text:
		marker = ""
		msgColor = ColorReset + ColorBlue
		argsColor = ColorReset + ColorWhite
		finfo = ""
		timeStamp = ""
	default:
		marker = ColorDefault + "---"
	}

	str := ColorReset
	for _, arg := range args {
		str += fmt.Sprintf("%s%v%s   ", argsColor, arg, ColorReset)
	}
	str = strings.Replace(str, "\n", "\n\t", -1)

	return fmt.Sprintf("%s %s %s   %s   %s",
		timeStamp,
		marker,
		msgColor+message+ColorReset,
		str,
		finfo+ColorReset)
}

func (p *logger) Debug(message string, args ...interface{}) {
	p.logToConsole(Debug, message, args...)
}

func (p *logger) Text(message string, args ...interface{}) {
	p.logToConsole(Text, message, args...)
}

func (p *logger) Info(message string, args ...interface{}) {
	p.logToConsole(Info, message, args...)
}

func (p *logger) Warning(message string, args ...interface{}) {
	p.logToConsole(Warning, message, args...)
}

func (p *logger) Error(message string, args ...interface{}) {
	p.logToConsole(Error, message, args...)
}

func (p *logger) Fatal(message string, args ...interface{}) {
	p.logToConsole(Fatal, message, args...)
}

func (p *logger) StartInline() {
	p.inline = true
}

func (p *logger) StopInline() {
	p.inline = false
	fmt.Println()
}

func SetGlobalLevel(lvl LogLevel) {
	globalLogger.GlobalLevel(lvl)
}

func ParseLevel(lvl string) LogLevel {
	switch lvl {
	case "debug":
		return Debug
	case "text":
		return Text
	case "info":
		return Info
	case "warning":
		return Warning
	case "error":
		return Error
	case "fatal":
		return Fatal
	default:
		return Text
	}
}

func LogToConsole(level LogLevel, message string, args ...interface{}) {
	globalLogger.logToConsole(level, message, args...)
}

func LogToFile(message string, args ...interface{}) {
	globalLogger.ToFile(message, args...)
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getWidth() int {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}
	return int(ws.Col)
}
