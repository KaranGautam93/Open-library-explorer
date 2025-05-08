package daemon

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"open-library-explorer/internal/models"
	"open-library-explorer/internal/utils"
	"time"
)

type LogExporter struct {
	Coll *mongo.Collection
}

func (l *LogExporter) InitLogExporter() {
	go func() {
		for {
			res, _ := l.Coll.Find(context.Background(), bson.M{"exported": false})

			var logs []models.AuditLog
			_ = res.All(context.Background(), &logs)

			if len(logs) > 0 {
				_ = utils.ExportData(logs)
				updateIds := []primitive.ObjectID{}

				for i := 0; i < len(logs); i++ {
					updateIds = append(updateIds, logs[i].ID)
				}

				l.Coll.UpdateMany(context.Background(), bson.M{"_id": bson.M{"$in": updateIds}}, bson.M{"$set": bson.M{"exported": true}})
			}
			time.Sleep(30 * time.Second)
		}
	}()
}
