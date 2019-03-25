package data

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	utils "github.com/aagoldingay/aye-go/utilities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// voter models voter document insert
type voter struct {
	ID             *primitive.ObjectID `bson:"_id,omitempty"`
	Username       string
	HasVoted       bool
	Password, Hash string
}

// VoterLogin declares whether voter logged in successfully and has voted
type VoterLogin struct {
	Success, HasVoted bool
	ID                string
}

// LoginVoter checks that a user has an account
func LoginVoter(username, password string, dbc *mongo.Client) (VoterLogin, error) {
	result := voter{}
	err := dbc.Database("aye-go").Collection("voter").
		FindOne(context.Background(), bson.M{"username": username}).Decode(&result)
	if err != nil {
		return VoterLogin{false, false, ""}, err
	}
	hashpass := md5.Sum([]byte(password + result.Hash))
	if result.Password != fmt.Sprintf("%x", hashpass) {
		return VoterLogin{false, false, ""}, nil
	}
	return VoterLogin{true, result.HasVoted, result.ID.Hex()}, nil
}

// Register adds new voters to the database
func Register(username, password string, dbc *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db := dbc.Database("aye-go")
	err := dbc.UseSession(ctx, func(sctx mongo.SessionContext) error {
		// START
		err := sctx.StartTransaction(options.Transaction())
		if err != nil {
			return err
		}

		// CODE
		salt := utils.GenerateCode(5)
		hashpass := md5.Sum([]byte(password + salt))
		_, err = db.Collection("voter").InsertOne(context.Background(),
			bson.D{{Key: "username", Value: username},
				{Key: "hasVoted", Value: false},
				{Key: "password", Value: fmt.Sprintf("%x", hashpass)},
				{Key: "hash", Value: salt}})

		if err != nil {
			return err
		}

		// COMMIT
		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// login - save active election id for comparison (accept / reject)
