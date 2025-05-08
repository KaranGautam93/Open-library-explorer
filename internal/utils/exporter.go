package utils

import (
	"fmt"
	"open-library-explorer/internal/models"
)

func ExportData(logs []models.AuditLog) error {
	for _, log := range logs {
		//change with actual calls
		fmt.Println(log.Timestamp, log.ID, log.Data)
	}
	return nil
}
