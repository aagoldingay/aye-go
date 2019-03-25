package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aagoldingay/aye-go/data"
	"github.com/gorilla/sessions"

	utils "github.com/aagoldingay/aye-go/utilities"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const alert = "[ALERT] : %v\n"
const startup = "[STARTING] : %v\n"
const stopping = "[STOPPING] : %v\n"
const usernameTMPL = "aye-go"

var (
	mdbClient       *mongo.Client
	sessionKey      = []byte{35, 250, 103, 131, 245, 255, 194, 76, 198, 188, 157, 217, 82, 104, 157, 5}
	store           *sessions.CookieStore
	currentElection data.Election
)

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

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		// first visit
		tmpl := template.Must(template.ParseFiles("static/tmpl/admin.html"))
		data := struct {
			Loggedin bool
			Enabled  string
		}{
			false,
			"",
		}
		tmpl.Execute(w, data)
	} else {
		session, _ := store.Get(r, "cookie-name")
		r.ParseForm()
		if r.FormValue("newelection") == "true" {
			// TODO election in progress (maybe return end date??)
			if admin, err := data.CheckAdmin(session.Values["id"].(string), mdbClient); !admin || err != nil {
				http.Error(w, "Problem or not admin", http.StatusTeapot)
				return
			}

			t := html.EscapeString(r.FormValue("title"))
			numOpts, _ := strconv.ParseInt(r.FormValue("numOptions"), 10, 32)
			opts := []string{}
			for i := 1; i <= int(numOpts); i++ {
				opts = append(opts, html.EscapeString(r.FormValue(
					fmt.Sprintf("option-%v", i))))
			}
			sd := r.FormValue("startdate")
			ed := r.FormValue("enddate")
			e, err := data.CreateElection(t, sd, ed, opts, mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem creating new election", http.StatusTeapot)
				return
			}
			currentElection = e

		} else {
			if r.FormValue("username") == "" || r.FormValue("password") == "" {
				http.Error(w, "Check credentials", http.StatusTeapot)
				return
			}
			resp, err := data.LoginAdmin(html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}
			if !resp.Success {
				http.Error(w, "Unsuccessful login", http.StatusTeapot)
			}

			session.Values["id"] = resp.ID
			session.Save(r, w)

			// reload page with loggedin = true
			tmpl := template.Must(template.ParseFiles("static/tmpl/admin.html"))

			// prevent form submission if currentElection populated and Now < enddate
			disabled := ""
			if !currentElection.EndDate.IsZero() || !time.Now().After(currentElection.EndDate) {
				disabled = "disabled"
			}
			data := struct {
				Loggedin bool
				Enabled  string
			}{
				true,
				disabled,
			}
			tmpl.Execute(w, data)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// default method = GET
	if r.Method == "POST" {
		r.ParseForm()
		if r.FormValue("method") == "register" {
			err := data.Register(html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}
			// TODO redirect to thanks (maybe with election start date?)
			tmpl := template.Must(template.ParseFiles("static/tmpl/thanks.html"))
			sd := "To be confirmed"
			if !currentElection.StartDate.IsZero() {
				sd = currentElection.StartDate.Format("2006-01-02")
			}
			data := struct {
				StartDate string
			}{
				sd,
			}
			tmpl.Execute(w, data)
		} else {
			session, _ := store.Get(r, "cookie-name")
			resp, err := data.LoginVoter(html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}
			if !resp.Success {
				http.Error(w, "Unsuccessful login", http.StatusTeapot)
			}
			if resp.HasVoted {
				http.Error(w, "Already voted", http.StatusTeapot)
			}

			session.Values["id"] = resp.ID
			session.Save(r, w)

			tmpl := template.Must(template.ParseFiles("static/tmpl/election.html"))
			sd := "To be confirmed"
			started := false
			if !currentElection.StartDate.IsZero() {
				if time.Now().After(currentElection.StartDate) {
					started = true
				}
				sd = currentElection.StartDate.Format("2006-01-02")
			}
			data := struct {
				Started   bool
				StartDate string
			}{
				started,
				sd,
			}
			tmpl.Execute(w, data)
		}
	} else {
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

	// get current election
	currentElection, err = data.GetCurrentElection(mdbClient)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			fmt.Printf(alert, err)
			os.Exit(1)
		}
	}

	store = sessions.NewCookieStore(sessionKey)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/admin", adminHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf(alert, fmt.Sprintf("failed to serve: %v", err))
		os.Exit(1)
	}
}
