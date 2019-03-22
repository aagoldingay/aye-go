package data

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Register adds new voters to the database
func Register(username, password string, db *mongo.Database) error {

	// TODO transaction
	// TODO password storing

	res, err := db.Collection("voter").InsertOne(context.Background(),
		bson.M{"username": username, "hasVoted": false, "password": password})

	if err != nil {
		return err
	}
	id := res.InsertedID
	fmt.Println(id)
	return nil
}

// login - save active election id for comparison (accept / reject)
