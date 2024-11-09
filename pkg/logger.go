package pkg

import (
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

func InitLogger(logPath string) {
	logFile := &lumberjack.Logger{
		Filename:   logPath, // Имя файла лога
		MaxSize:    3,       // Максимальный размер файла в мегабайтах
		MaxBackups: 3,       // Максимальное количество резервных копий
		MaxAge:     28,      // Максимальный срок хранения резервных копий (в днях)
		Compress:   true,    // Сжать резервные копии
	}

	logrus.SetOutput(logFile)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}
