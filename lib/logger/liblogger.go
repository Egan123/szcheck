package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

var expireDuration time.Duration
var expireTime = "30m"               //日志卷动的间隔时间 30 分钟
var logRotageSize = int64(100000000) //日志卷动的文件大小 100MB

const (
	LEVEL_EMERG = iota
	LEVEL_ALERT
	LEVEL_CRIT
	LEVEL_ERR
	LEVEL_WARNING
	LEVEL_NOTICE
	LEVEL_INFO
	LEVEL_DEBUG
)

type loggerStruct struct {
	Logger       *log.Logger
	Level        int
	Flag         int
	File         *os.File
	FileDir      string
	FileName     string
	FullFileName string
	ExtType      string
}

var logger loggerStruct
var timeLayout = "20060102_1504"

var loggerReqPrintln chan string
var loggerReqPrintf chan string
var loggerReqPanic chan string

//logger for ExtType
const (
	EXT_LOGGER_MAX = 5 //允许注册的最大扩展日志数量
)

var extLoggerRegMax uint //已注册的扩展日志数量
var extLogger [EXT_LOGGER_MAX]loggerStruct

type extLoggerReqStruct struct {
	ExtType uint
	String  string
}

var extLoggerReqChan chan *extLoggerReqStruct

func init() {
	logger.Logger = nil
	logger.Flag = log.Ldate | log.Ltime
	logger.Logger = log.New(os.Stdout, "", logger.Flag)
	logger.Level = LEVEL_DEBUG

	loggerReqPrintln = make(chan string, 10000)
	loggerReqPrintf = make(chan string, 1000)
	loggerReqPanic = make(chan string, 10)

	extLoggerRegMax = uint(0)
	extLoggerReqChan = make(chan *extLoggerReqStruct, 1000)

	expireDuration, _ = time.ParseDuration("30m") //初始化周期时间，避免未 InitLoggerXXX() 的时候影响定时器工作
	go goSaveLogger()
}

func InitLogger(fileDir string, fileName string, logLevel string) bool {
	logger.FileDir = fileDir
	logger.FileName = fileName
	SetLevelString(logLevel)

	if !createLoggerFile() {
		fmt.Println("logger.InitLogger() create logger file fail")
		return false
	}

	var err error
	expireDuration, err = time.ParseDuration(expireTime)
	if err != nil {
		fmt.Println(fmt.Sprintf("logger.InitLogger() ParseDuration(%s) error", expireTime))
		return false
	}

	return true
}

func InitExtLogger(extType uint, extTypeStr string) bool {
	if extType >= EXT_LOGGER_MAX {
		Error("logger.InitLoggerExt() logger ext type so big, max:", EXT_LOGGER_MAX)
		return false
	}

	extLogger[extType].ExtType = extTypeStr

	if !createExtLoggerFile(extType) {
		fmt.Println("logger.createExtLoggerFile() create logger file fail")
		return false
	}

	extLogger[extType].ExtType = extTypeStr
	extLoggerRegMax++

	return true
}

func goSaveLogger() {
	var loggerReq string
	var extLoggerReq *extLoggerReqStruct

	for {
		select {
		case loggerReq = <-loggerReqPrintln:
			logger.Logger.Println(loggerReq)

		case loggerReq = <-loggerReqPrintf:
			logger.Logger.Printf(loggerReq)

		case loggerReq = <-loggerReqPanic:
			logger.Logger.Panicln(loggerReq)

		case extLoggerReq = <-extLoggerReqChan:
			extLogger[extLoggerReq.ExtType].Logger.Println(extLoggerReq.String)

		case <-time.After(expireDuration):
			if logger.File != nil {
				fStat, err := logger.File.Stat()
				if err == nil {
					if fStat.Size() >= logRotageSize {
						Notice("logger.goSaveLogger() logRotage")
						createLoggerFile()
					}
				} else {
					Error("logger.goSaveLogger() get file stat error(fileLogger)", err)
				}
			}

			//检查扩展日志
			if extLoggerRegMax >= 1 {
				for extType := uint(0); extType <= EXT_LOGGER_MAX-1; extType++ {
					if extLogger[extType].File != nil {
						fStat, err := extLogger[extType].File.Stat()
						if err == nil {
							if fStat.Size() >= logRotageSize {
								Notice("logger.goSaveLogger() logRotage")
								createExtLoggerFile(extType)
							}
						} else {
							Error("logger.goSaveLogger() get file stat error(fileExtLogger)", err)
						}
					}
				}
			}
		}
	}
}

func createLoggerFile() (ok bool) {
	var err error

	logger.FullFileName = fmt.Sprintf("%s/%s_%s.log", logger.FileDir, logger.FileName, time.Now().Format(timeLayout))
	logger.File, err = os.OpenFile(logger.FullFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		Error(err)
		return false
	}

	logger.Logger = log.New(logger.File, "", logger.Flag)

	return true
}

func createExtLoggerFile(extType uint) (ok bool) {
	var err error

	extLogger[extType].FullFileName = fmt.Sprintf("%s/%s_%s_%s.log", logger.FileDir, logger.FileName, extLogger[extType].ExtType, time.Now().Format(timeLayout))
	extLogger[extType].File, err = os.OpenFile(extLogger[extType].FullFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		Error(err)
		return false
	}

	extLogger[extType].Logger = log.New(extLogger[extType].File, "", logger.Flag)

	return true
}

func SetLevelString(newLevelString string) bool {
	newLevel := LEVEL_DEBUG
	switch newLevelString {
	case "DEBUG":
		newLevel = LEVEL_DEBUG
	case "INFO":
		newLevel = LEVEL_INFO
	case "NOTICE":
		newLevel = LEVEL_NOTICE
	case "WARNING":
		newLevel = LEVEL_WARNING
	case "ERROR":
		newLevel = LEVEL_ERR
	default:
		Error("logger.SetLevel(): bad new level string:", newLevelString)
		return false
	}

	SetLevel(newLevel)
	Notice("logger.SetLevel(): change to level[", newLevelString, "] ok")
	return true
}

func SetLevel(newLevel int) bool {
	logger.Level = newLevel
	return true
}

func GetLevel() int {
	return logger.Level
}

func Debug(v ...interface{}) {
	if logger.Level >= LEVEL_DEBUG {
		loggerReqPrintln <- fmt.Sprint("[DEBUG]", v)
	}
}

func Info(v ...interface{}) {
	if logger.Level >= LEVEL_INFO {
		loggerReqPrintln <- fmt.Sprint("[INFO]", v)
	}
}

func Notice(v ...interface{}) {
	if logger.Level >= LEVEL_NOTICE {
		loggerReqPrintln <- fmt.Sprint("[NOTICE]", v)
	}
}

func Warning(v ...interface{}) {
	if logger.Level >= LEVEL_ERR {
		loggerReqPrintln <- fmt.Sprint("[WARNING]", v)
	}
}

func Error(v ...interface{}) {
	if logger.Level >= LEVEL_ERR {
		loggerReqPrintln <- fmt.Sprint("[ERROR]", v)
	}
}

func Panic(v ...interface{}) {
	loggerReqPanic <- fmt.Sprint("[Panic]", v)
}

func Printf(format string, v ...interface{}) {
	loggerReqPrintf <- fmt.Sprintf(format, v...)
}

func ExtLog(extType uint, v ...interface{}) {
	req := new(extLoggerReqStruct)
	req.ExtType = extType
	req.String = fmt.Sprint(v)

	extLoggerReqChan <- req
}
