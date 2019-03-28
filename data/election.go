package data

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Election models election document insert
type Election struct {
	ID                 *primitive.ObjectID `bson:"_id,omitempty"`
	Title              string
	StartDate, EndDate time.Time
	Options            []string
}

type election struct {
	ID        *primitive.ObjectID `bson:"_id,omitempty"`
	Title     string              `bson:"title"`
	StartDate int64               `bson:"start-date"`
	EndDate   int64               `bson:"end-date"`
	Options   []string            `bson:"options"`
}

type result struct {
	Identifier string
	option     string
	coerced    bool
}

// AddResult adds the selected voter preference to current election
func AddResult(voterID, electionID, info1, info2, option string, dbc *mongo.Client) (bool, error) {
	// u, _ := primitive.ObjectIDFromHex(voterId)
	// e, _ := primitive.ObjectIDFromHex(electionId)

	// hash info1 and 2

	// update results with has and selected option

	return false, nil
}

// CreateElection parses form input and adds to the database
func CreateElection(title, startdate, enddate string, opts []string, dbc *mongo.Client) (Election, error) {
	// parse dates
	sdp, _ := time.Parse("2006-01-02", startdate)
	sd := sdp.Unix()
	edp, _ := time.Parse("2006-01-02", enddate)
	ed := edp.Unix()

	// compile document
	doc := bson.D{{Key: "title", Value: title},
		{Key: "start-date", Value: sd},
		{Key: "end-date", Value: ed},
		{Key: "options", Value: opts}}

	var e Election

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
		res, err := db.Collection("election").InsertOne(context.Background(), doc)
		if err != nil {
			return err
		}

		// COMMIT
		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return err
		}
		if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
			e = Election{ID: &oid, Title: title, StartDate: sdp, EndDate: edp}
		}
		return nil
	})
	if err != nil {
		return Election{}, err
	}
	return e, nil
}

// GetCurrentElection accesses the database to find the next election
func GetCurrentElection(dbc *mongo.Client) (Election, error) {
	res := election{}
	y, m, d := time.Now().Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Now().Location())

	err := dbc.Database("aye-go").Collection("election").FindOne(context.Background(),
		bson.D{{Key: "end-date",
			Value: bson.D{{Key: "$gte", Value: start.Unix()}},
		}}).Decode(&res)
	if err != nil {
		return Election{}, err
	}
	if res.Title == "" || res.EndDate == 0 || res.StartDate == 0 || len(res.Options) == 0 {
		return Election{}, errors.New("no info returned from GetCurrentElection")
	}

	return Election{ID: res.ID, Title: res.Title,
		StartDate: time.Unix(res.StartDate, 0),
		EndDate:   time.Unix(res.EndDate, 0), Options: res.Options}, nil
}
