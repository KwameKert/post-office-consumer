package log

import (
	cxt "context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type logLayer struct {
	collection *mongo.Collection
}

func NewLogRepoLayer(db *mongo.Database) *logLayer {
	return &logLayer{
		collection: db.Collection("Logs"),
	}
}

func (al *logLayer) Create(log *Log) error {
	log.CreatedAt = time.Now()
	log.UpdatedAt = time.Now()
	_, err := al.collection.InsertOne(cxt.TODO(), &log)
	if err != nil {
		return err
	}
	return nil
}
