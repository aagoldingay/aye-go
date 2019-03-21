package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	utils "github.com/aagoldingay/aye-go/utilities"
	"github.com/globalsign/mgo"
)

const alert = "[ALERT] : %v\n"
const startup = "[STARTING] : %v\n"
const stopping = "[STOPPING] : %v\n"
const usernameTMPL = "aye-go"

// Voter models voter document insert
type Voter struct {
	ID              int
	ShortIdentifier string
	HasVoted        bool
}

// Election models election document insert
type Election struct {
	ID                 int
	StartDate, EndDate time.Time
	Options            []string
	IntegrationFormat  string
}

var db *mgo.Database

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
			if md5.Sum([]byte(r.FormValue("password"))) != md5.Sum([]byte(r.FormValue("confirmpassword"))) {
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}
			// TODO register
		} else {
			// TODO login
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
	sess, err := mgo.Dial("localhost:27017")
	if err != nil && err.Error() == "no reachable servers" {
		fmt.Printf(alert, "mongodb not installed")
		os.Exit(1)
	}

	utils.Setup(-1)
	d := sess.DB("ayedb")
	db = d

	data, err := readConfig()
	addNewUser := false
	if err != nil {
		if err.Error() == "new user" {
			addNewUser = true
		} else {
			fmt.Printf(alert, err)
			os.Exit(1)
		}
	}
	if len(data) != 2 {
		fmt.Printf(alert, "login data not as expected")
		os.Exit(1)
	}

	// successful config file
	if addNewUser {
		err = db.UpsertUser(&mgo.User{Username: data[0], Password: data[1]})
		if err != nil {
			fmt.Printf(alert, err)
		}
	}
	err = db.Login(data[0], data[1])
	if err != nil {
		fmt.Printf(alert, err)
	}
	fmt.Println(data[0])
	fmt.Println(data[1])

	// fmt.Printf(alert, db.Name)
	// fmt.Println("aye-go")

	// start http service
	fmt.Printf(startup, "http thread")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.HandleFunc("/", indexHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf(alert, fmt.Sprintf("failed to serve: %v", err))
		os.Exit(1)
	}
	fmt.Printf(stopping, "shutting down HTTP")
}
