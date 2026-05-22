package db

import (
	"context"
	"testing"
	"time"
)

func TestOpenRejectsEmptyDatabaseURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pool, err := Open(ctx, "")
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("Open returned nil error for empty database URL")
	}
}
