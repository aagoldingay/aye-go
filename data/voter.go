package data

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Voter models voter document insert
type Voter struct {
	ID       int
	Username string
	HasVoted bool
	Password string
}

// LoginVoter checks that a user has an account
func LoginVoter(username, password string, db *mongo.Database) (bool, error) {
	// TODO transaction
	// TODO md5
	result := Voter{}
	err := db.Collection("voter").FindOne(context.Background(), bson.M{"username": username}).Decode(&result)
	if err != nil {
		return false, err
	}
	if result.Password != password {
		return false, nil
	}
	return true, nil
}

// Register adds new voters to the database
func Register(username, password string, db *mongo.Database) error {

	// TODO transaction
	// TODO password storing (md5)

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
