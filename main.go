package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aagoldingay/aye-go/data"

	utils "github.com/aagoldingay/aye-go/utilities"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const alert = "[ALERT] : %v\n"
const startup = "[STARTING] : %v\n"
const stopping = "[STOPPING] : %v\n"
const usernameTMPL = "aye-go"

// Election models election document insert
type Election struct {
	ID                 int
	StartDate, EndDate time.Time
	Options            []string
	IntegrationFormat  string
}

var mdbClient *mongo.Client

func readConfig() ([]string, error) {
	// initial read
	b, err := ioutil.ReadFile("config.txt")
	if err != nil { // file not found
		fmt.Printf(alert, err)
		data, err := writeConfig()
		if err != nil {
			return []string{}, err
		}
		return data, errors.New("new user")
	}

	// successful read
	contents := string(b)
	lines := strings.Split(contents, "\n")
	if len(lines) < 2 || !strings.Contains(lines[0], usernameTMPL) { // improper file
		data, err := writeConfig()
		if err != nil {
			return []string{}, err
		}
		return data, errors.New("new user")
	}

	// proper file read
	return lines, nil
}

func writeConfig() ([]string, error) {
	fmt.Printf(alert, "writing new config file")

	// configure new username, password
	u := usernameTMPL + utils.GenerateCode(5)
	p := utils.GenerateCode(8)
	data := []string{u, p}

	// write to config.txt
	contents := []byte(fmt.Sprintf("%v\n%v", u, p))

	err := ioutil.WriteFile("config.txt", contents, 0644)
	if err != nil {
		fmt.Printf(alert, err)
		return []string{}, errors.New("problem on write") // return empty, since failed write means unrecoverable information after a shutdown
	}
	return data, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// default method = GET
	if r.Method == "POST" {
		// write to db
		r.ParseForm()
		if r.FormValue("method") == "register" {
			// TODO register
			err := data.Register(html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}
			// redirect to thanks (maybe with election start date?)
		} else {
			// TODO login
			resp, err := data.LoginVoter(html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}
			fmt.Printf(alert, resp)
			if !resp.Success {
				http.Error(w, "Unsuccessful login", http.StatusTeapot)
			}
			if resp.HasVoted {
				http.Error(w, "Already voted", http.StatusTeapot)
			}
			// fmt.Printf(alert, resp.ObjectID)
			// TODO start session
		}
	} else {
		// register to vote page
		// login info, register to vote button
		// use js to generate code? then submits to post with password, on success = show shorthand id
		tmpl := template.Must(template.ParseFiles("static/tmpl/index.html"))
		tmpl.Execute(w, nil)
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		fmt.Printf(alert, fmt.Sprintf("connect err = %v", err))
		os.Exit(1)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Printf(alert, fmt.Sprintf("ping err = %v", err))
		os.Exit(1)
	}
	mdbClient = client
	//db = client.Database("aye-go")

	utils.Setup(-1)

	// data, err := readConfig()
	// addNewUser := false
	// if err != nil {
	// 	if err.Error() == "new user" {
	// 		addNewUser = true
	// 	} else {
	// 		fmt.Printf(alert, err)
	// 		os.Exit(1)
	// 	}
	// }
	// if len(data) != 2 {
	// 	fmt.Printf(alert, "login data not as expected")
	// 	os.Exit(1)
	// }

	// start http service
	fmt.Printf(startup, "http thread")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.HandleFunc("/", indexHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf(alert, fmt.Sprintf("failed to serve: %v", err))
		os.Exit(1)
	}
}
