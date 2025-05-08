package utils

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"open-library-explorer/internal/models"
)

type Logger struct {
	Collection *mongo.Collection
}

func (l *Logger) Log(ctx context.Context, entity, action string, data any) error {
	log := models.AuditLog{
		Timestamp: time.Now(),
		Entity:    entity,
		Action:    action,
		Data:      data,
	}
	_, err := l.Collection.InsertOne(ctx, log)
	return err
}
