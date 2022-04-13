package utils
import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"log"
	"fmt"
)
var Logger *zap.Logger
/*
This is the initializeation function that needs to be invoked
at the startup of the server. 
*/
func InitializeLogger(logDir string, fileName string) {
	loggingFilePath := ""
	
	config := zap.NewProductionEncoderConfig()
	if logDir == "" {
		homeDir, _ := os.UserHomeDir();
		loggingFilePath = homeDir
	} else {
		loggingFilePath = logDir
	}
	if fileName == "" {
		loggingFilePath += "/Spinup.log"
	} else {
		loggingFilePath += "/" + fileName
	}

	log.Println(fmt.Sprintf("Spinup Logging file -> %s\n", loggingFilePath))
    config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	consoleEncoder := zapcore.NewConsoleEncoder(config)
	logFile, _ := os.OpenFile(loggingFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel),
	)
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}