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
	ID                       *primitive.ObjectID `bson:"_id,omitempty"`
	Username                 string
	HasVoted                 bool
	Password, Hash, Safeword string
}

// VoterLogin declares whether voter logged in successfully and has voted
type VoterLogin struct {
	Success, HasVoted bool
	ID                string
}

// VoterVote stores information on whether the user has voted or was coerced (via safeword)
type VoterVote struct {
	HasVoted, Coerced bool
}

// CheckVoter determines whether a voter is registered and eligible to vote
func CheckVoter(userID, safeword string, dbc *mongo.Client) (VoterVote, error) {
	u, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return VoterVote{true, false}, err
	}

	r := struct {
		HasVoted bool   `bson:"hasVoted"`
		Safeword string `bson:"safeword"`
		Hash     string `bson:"hash"`
	}{}
	err = dbc.Database("aye-go").Collection("voter").FindOne(context.Background(), bson.M{"_id": u}).Decode(&r)

	// cause for rejection
	if r.HasVoted || err == mongo.ErrNoDocuments {
		return VoterVote{r.HasVoted, false}, nil
	}

	// unintended error
	if err != nil {
		return VoterVote{r.HasVoted, false}, err
	}

	hsw := md5.Sum([]byte(safeword + r.Hash))

	// all clear
	return VoterVote{r.HasVoted, fmt.Sprintf("%x", hsw) != r.Safeword}, nil
}

// LoginVoter checks that a user has an account
func LoginVoter(electionID, username, password, safeword string, dbc *mongo.Client) (VoterLogin, error) {
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

	if !result.HasVoted { // check safeword
		res, err := GetOneResult(electionID, username, safeword, dbc)
		if err != nil {
			return VoterLogin{false, false, ""}, nil
		}
		if len(res) > 0 { //username+safeword has been used - "has voted" (was coerced, so fake thiss)
			result.HasVoted = true
		}
	}
	return VoterLogin{true, result.HasVoted, result.ID.Hex()}, nil
}

// Register adds new voters to the database
func Register(username, password, safeword string, dbc *mongo.Client) error {
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
		hashsafeword := md5.Sum([]byte(safeword + salt))
		_, err = db.Collection("voter").InsertOne(context.Background(),
			bson.D{{Key: "username", Value: username},
				{Key: "hasVoted", Value: false},
				{Key: "password", Value: fmt.Sprintf("%x", hashpass)},
				{Key: "safeword", Value: fmt.Sprintf("%x", hashsafeword)},
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
