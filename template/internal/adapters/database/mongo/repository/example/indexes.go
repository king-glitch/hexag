package example

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// EnsureIndexes is empty until you have a query pattern that needs one — add
// db.Collection(...).Indexes().CreateMany(...) here then, following the
// shape used by other repositories once you have more than one.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	return nil
}
