package log

import (
	"github.com/ayufan/golang-kardianos-service"
	"github.com/sirupsen/logrus"
)

type systemLogger interface {
	service.Logger
}

type systemService interface {
	service.Service
}

type SystemServiceLogHook struct {
	systemLogger
	Level logrus.Level
}

func (s *SystemServiceLogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}

func (s *SystemServiceLogHook) Fire(entry *logrus.Entry) error {
	if entry.Level > s.Level {
		return nil
	}

	msg, err := entry.String()
	if err != nil {
		return err
	}

	switch entry.Level {
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		s.Error(msg)
	case logrus.WarnLevel:
		s.Warning(msg)
	case logrus.InfoLevel:
		s.Info(msg)
	}

	return nil
}

func SetSystemLogger(logrusLogger *logrus.Logger, svc systemService) {
	logger, err := svc.SystemLogger(nil)

	if err == nil {
		hook := new(SystemServiceLogHook)
		hook.systemLogger = logger
		hook.Level = logrus.GetLevel()

		logrusLogger.AddHook(hook)
	} else {
		logrusLogger.WithError(err).Error("Error while setting up the system logger")
	}
}
