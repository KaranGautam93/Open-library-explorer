package utils

import (
	"context"
	"fmt"
)

func AppendToEmailLog(ctx context.Context, memberID string, barcode string) {
	fmt.Printf("[EMAIL LOG] Notified member %s about reserved copy %s\n", memberID, barcode)
}
