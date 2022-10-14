package log

import (
	log "github.com/sirupsen/logrus"
)

type LogServiceLayer struct {
	repository logLayer
}

func NewLogServiceLayer(r logLayer) *LogServiceLayer {
	return &LogServiceLayer{
		repository: r,
	}
}
func (service *LogServiceLayer) AddLog(data Log) {
	if err := service.repository.Create(&data); err != nil {
		log.Error(err)
	}

	log.Info("Log saved successfully")
}
