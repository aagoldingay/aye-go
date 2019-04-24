package data

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
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
	CoercionRes        bool `bson:"coercion"`
	Options            []string
}

// results models returned information from database
type results struct {
	ID     *primitive.ObjectID `bson:"_id,omitempty"`
	Result []Result
}

// Result models a single result when being returned for use
type Result struct {
	Identifier string `bson:"identifier"`
	Option     string `bson:"option"`
}

type election struct {
	ID          *primitive.ObjectID `bson:"_id,omitempty"`
	Title       string              `bson:"title"`
	StartDate   int64               `bson:"start-date"`
	EndDate     int64               `bson:"end-date"`
	CoercionRes bool                `bson:"coercion"`
	Options     []string            `bson:"options"`
}

type result struct {
	Identifier string
	option     string
	coerced    bool
}

// AddResult adds the selected voter preference to current election
func AddResult(voterID, electionID, info1, info2, option string, coerced bool, dbc *mongo.Client) (bool, error) {
	u, _ := primitive.ObjectIDFromHex(voterID)
	e, _ := primitive.ObjectIDFromHex(electionID)

	// hash info1 and 2
	newID := fmt.Sprintf("%x", md5.Sum([]byte(info1+info2)))

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
		// update results with has and selected option
		_, err = db.Collection("election").UpdateOne(context.Background(),
			bson.M{"_id": e},
			bson.D{
				{Key: "$push", Value: bson.D{
					{Key: "result", Value: bson.D{
						{Key: "identifier", Value: newID},
						{Key: "option", Value: option},
						{Key: "coerced", Value: coerced},
					}},
				}}})

		if err != nil {
			return err
		}

		// update voter (hasVoted = true)
		_, err = db.Collection("voter").UpdateOne(context.Background(),
			bson.M{"_id": u},
			bson.M{"$set": bson.M{"hasVoted": !coerced}})

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
		return false, err
	}

	return true, nil
}

// GetOneResult returns a result of the corresponding voter
func GetOneResult(electionID, username, safeword string, dbc *mongo.Client) ([]Result, error) {
	e, _ := primitive.ObjectIDFromHex(electionID)
	voter := fmt.Sprintf("%x", md5.Sum([]byte(username+safeword)))

	cur, err := dbc.Database("aye-go").Collection("election").Aggregate(context.Background(), []bson.M{
		bson.M{"$match": bson.M{"_id": e}},                                            // election
		bson.M{"$unwind": bson.M{"path": "$result"}},                                  // separate array of results
		bson.M{"$match": bson.M{"result.identifier": voter}},                          // find result matching username+safeword identifier
		bson.M{"$group": bson.M{"_id": "$_id", "result": bson.M{"$push": "$result"}}}, // regroup array
	})
	if err != nil {
		return []Result{}, err
	}

	r := results{}
	for cur.Next(context.Background()) {
		cur.Decode(&r)
	}

	cur.Close(context.Background())
	if err := cur.Err(); err != nil {
		return []Result{}, err
	}
	return r.Result, nil
}

// GetResults returns all results for an election
func GetResults(electionID string, dbc *mongo.Client) ([]Result, error) {
	e, _ := primitive.ObjectIDFromHex(electionID)

	cur, err := dbc.Database("aye-go").Collection("election").Aggregate(context.Background(), []bson.M{
		bson.M{"$match": bson.M{"_id": e}},                                            // election
		bson.M{"$unwind": bson.M{"path": "$result"}},                                  // separate array of results
		bson.M{"$match": bson.M{"result.coerced": false}},                             // find only valid results
		bson.M{"$group": bson.M{"_id": "$_id", "result": bson.M{"$push": "$result"}}}, // regroup array
	})
	if err != nil {
		return []Result{}, err
	}

	r := results{}
	for cur.Next(context.Background()) {
		cur.Decode(&r)
	}

	cur.Close(context.Background())
	if err := cur.Err(); err != nil {
		return []Result{}, err
	}

	return r.Result, nil
}

// CreateElection parses form input and adds to the database
func CreateElection(title, startdate, enddate string, opts []string, coercion bool, dbc *mongo.Client) (Election, error) {
	// parse dates
	sdp, _ := time.Parse("2006-01-02", startdate)
	sd := sdp.Unix()
	edp, _ := time.Parse("2006-01-02", enddate)
	ed := edp.Unix()

	// compile document
	doc := bson.D{{Key: "title", Value: title},
		{Key: "start-date", Value: sd},
		{Key: "end-date", Value: ed},
		{Key: "coercion", Value: coercion},
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
			e = Election{ID: &oid, Title: title, StartDate: sdp, EndDate: edp, CoercionRes: coercion, Options: opts}
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
		StartDate:   time.Unix(res.StartDate, 0),
		EndDate:     time.Unix(res.EndDate, 0),
		CoercionRes: res.CoercionRes, Options: res.Options}, nil
}
