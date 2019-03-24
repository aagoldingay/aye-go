package data

import (
	"context"
	"crypto/md5"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
