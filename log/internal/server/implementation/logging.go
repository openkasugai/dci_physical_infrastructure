// Copyright 2026 NTT, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package implementation

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"

	"log_module/internal/server/interfaces" // import for interface
)

type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(time.RFC3339)
	keyId := entry.Data["keyId"]
	json := entry.Data["json"]

	logMessage := fmt.Sprintf("%s %s %s\n", timestamp, keyId, json)
	return []byte(logMessage), nil
}

// struct of Logging
type LoggingImplement struct {
	Logger           klog.Logger
	LumberJackLogger *lumberjack.Logger
	Logrus           *logrus.Logger
}

func (l *LoggingImplement) Init(loggingConfig interfaces.LoggingConfig) (err error) {
	l.Logger.V(2).Info("start Init", "loggingConfig", loggingConfig)
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing logging system")

	// logging setting
	logFileName := fmt.Sprintf("%s_%s.log", loggingConfig.LogFile, time.Now().Format("20060102"))
	logFilePath := loggingConfig.LogPath

	l.Logger.V(2).Info("log file configuration", "fileName", logFileName, "filePath", logFilePath)

	l.LumberJackLogger = &lumberjack.Logger{
		Filename:   logFilePath + "/" + logFileName, // path of log-file
		MaxSize:    loggingConfig.MaxSize,           // max size(MB) of log-file
		MaxBackups: loggingConfig.MaxBackups,        // max num of backup
		MaxAge:     loggingConfig.MaxAge,            // max age days
		Compress:   false,                           // compress flag
	}

	l.Logger.V(2).Info("lumberjack logger configured", "maxSize", loggingConfig.MaxSize, "maxBackups", loggingConfig.MaxBackups, "maxAge", loggingConfig.MaxAge)

	l.Logrus = logrus.New()
	l.Logrus.SetOutput(l.LumberJackLogger)
	l.Logrus.SetLevel(logrus.InfoLevel)
	l.Logrus.SetFormatter(new(CustomFormatter))

	l.Logger.V(2).Info("branch: logrus configured successfully")
	l.Logger.Info("logging system initialization completed")
	return nil
}

func (l *LoggingImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	l.Logger.Info("finalizing logging system")

	if l.LumberJackLogger != nil {
		l.Logger.V(2).Info("branch: closing lumberjack logger")
		_ = l.LumberJackLogger.Close()
	} else {
		l.Logger.V(2).Info("branch: lumberjack logger is nil")
	}
}

func (l *LoggingImplement) Write(keyId string, json string) (err error) {
	l.Logger.V(2).Info("start Write", "keyId", keyId)
	defer func() {
		l.Logger.V(2).Info("end Write", "err", err)
	}()

	if l.Logrus != nil {
		l.Logger.V(2).Info("branch: writing log entry", "keyId", keyId)
		l.Logrus.WithFields(logrus.Fields{
			"keyId": keyId,
			"json":  json,
		}).Info("")
	} else {
		l.Logger.V(2).Info("branch: logrus is nil, cannot write log")
	}

	return nil
}
