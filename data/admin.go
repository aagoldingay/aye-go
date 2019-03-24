package data

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// admin models admin document
type admin struct {
	Username, Password, Hash string
}

// AdminLogin declares whether admin loggin in successfully
type AdminLogin struct {
	Success  bool
	Username string
}

// Election models election document insert
type Election struct {
	Title              string
	StartDate, EndDate int64
	Options            []string
}

// CreateElection parses form input and adds to the database
func CreateElection(title, startdate, enddate string, opts []string, dbc *mongo.Client) error {
	// parse dates
	d, _ := time.Parse("2006-01-02", startdate)
	sd := d.Unix()
	d, _ = time.Parse("2006-01-02", enddate)
	ed := d.Unix()
	//sd, _ := strconv.ParseInt(startdate, 10, 64)
	//ed, _ := strconv.ParseInt(enddate, 10, 64)
	// compile document
	doc := bson.M{"title": title, "start-date": sd, "end-date": ed, "options": opts}

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
		_, err = db.Collection("election").InsertOne(context.Background(), doc)
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

// LoginAdmin checks that an admin typed the correct credentials to their corresponding account
func LoginAdmin(username, password string, dbc *mongo.Client) (AdminLogin, error) {
	result := admin{}
	err := dbc.Database("aye-go").Collection("admin").
		FindOne(context.Background(), bson.M{"username": username}).Decode(&result)
	if err != nil {
		return AdminLogin{false, ""}, err
	}
	hashpass := md5.Sum([]byte(password + result.Hash))
	if result.Password != fmt.Sprintf("%x", hashpass) {
		return AdminLogin{false, ""}, nil
	}
	return AdminLogin{true, result.Username}, nil
}
