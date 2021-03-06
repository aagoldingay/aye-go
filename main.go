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
	"github.com/kabukky/httpscerts"

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
			enableCoercion := utils.ConvertCheckbox(r.FormValue("coercion"))
			t := html.EscapeString(r.FormValue("title"))
			numOpts, _ := strconv.ParseInt(r.FormValue("numOptions"), 10, 32)
			opts := []string{}
			for i := 1; i <= int(numOpts); i++ {
				opts = append(opts, html.EscapeString(r.FormValue(
					fmt.Sprintf("option-%v", i))))
			}
			sd := r.FormValue("startdate")
			ed := r.FormValue("enddate")
			e, err := data.CreateElection(t, sd, ed, opts, enableCoercion, mdbClient)
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
				return
			}
			if !resp.Success {
				http.Error(w, "Unsuccessful login", http.StatusTeapot)
				return
			}

			session.Values["id"] = resp.ID
			session.Save(r, w)

			// reload page with loggedin = true
			tmpl := template.Must(template.ParseFiles("static/tmpl/admin.html"))

			// prevent form submission if currentElection populated and Now < enddate
			disabled := ""
			ended := false
			if !currentElection.EndDate.IsZero() || !time.Now().After(currentElection.EndDate) {
				disabled = "disabled"
			}
			if !currentElection.EndDate.IsZero() && time.Now().After(currentElection.EndDate) {
				ended = true
			}
			data := struct {
				Loggedin bool
				Enabled  string
				Ended    bool
			}{
				true,
				disabled,
				ended,
			}
			tmpl.Execute(w, data)
			return
		}
	}
}

func auditHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")
	if admin, err := data.CheckAdmin(session.Values["id"].(string), mdbClient); !admin || err != nil {
		http.Error(w, "Problem or not admin", http.StatusTeapot)
		return
	}

	res, err := data.GetAllResults(currentElection.ID.Hex(), mdbClient)
	if err != nil {
		fmt.Printf(alert, err)
		http.Error(w, "Problem encountered", http.StatusTeapot)
		return
	}

	tmpl := template.Must(template.ParseFiles("static/tmpl/audit.html"))
	data := struct {
		Title   string
		Options string
		Results []data.FullResult
	}{
		Title:   currentElection.Title,
		Options: strings.Join(currentElection.Options, ", "),
		Results: res,
	}
	tmpl.Execute(w, data)
	return
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")
	session.Values["id"] = ""
	session.Save(r, w)
	http.Error(w, "Logged out", http.StatusOK)
}

func pubRecordHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("static/tmpl/live.html"))
	req := r.Method == "POST"
	sd := "To be confirmed"
	ed := "To be confirmed"
	started := false
	if !currentElection.StartDate.IsZero() {
		if time.Now().After(currentElection.StartDate) {
			started = true
		}
		sd = currentElection.StartDate.Format("2006-01-02")
		ed = currentElection.EndDate.Format("2006-01-02")
	}
	pageData := struct {
		Started, Requested bool
		StartDate          string
		Title              string
		EndDate            string
		Results            []data.Result
	}{
		Started:   started,
		Requested: req,
		StartDate: sd,
		Title:     currentElection.Title,
		EndDate:   ed,
	}

	if req {
		// get one or all
		r.ParseForm()
		if r.FormValue("username") != "" {
			results, err := data.GetOneResult(currentElection.ID.Hex(), html.EscapeString(r.FormValue("username")), html.EscapeString(r.FormValue("safeword")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
				return
			}
			pageData.Results = results
			goto Execute
		}
		results, err := data.GetResults(currentElection.ID.Hex(), mdbClient)
		if err != nil {
			fmt.Printf(alert, err)
			http.Error(w, "Problem occurred", http.StatusTeapot)
			return
		}
		pageData.Results = results
	}
Execute:
	tmpl.Execute(w, pageData)

}

func submitVoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()

		// confirm electionID hasnt changed - reject if true
		if r.FormValue("election") != currentElection.ID.Hex() {
			http.Error(w, "Problem occured", http.StatusTeapot)
			return
		}

		// confirm voter is a voter
		session, _ := store.Get(r, "cookie-name")
		voter, err := data.CheckVoter(session.Values["id"].(string), r.FormValue("safeword"), mdbClient)
		if voter.HasVoted {
			if err != nil {
				fmt.Printf(alert, err)
			}
			http.Error(w, "No authorisation", http.StatusTeapot)
			return
		}

		ok, err := data.AddResult(session.Values["id"].(string), currentElection.ID.Hex(), r.FormValue("username"), r.FormValue("safeword"), r.FormValue("option"), voter.Coerced, mdbClient)
		if err != nil {
			fmt.Printf(alert, err)
			http.Error(w, "Problem occurred", http.StatusTeapot)
			return
		}
		if !ok {
			http.Error(w, "Problem occurred", http.StatusTeapot)
			return
		}

		// tmpl thanks
		http.Error(w, "Thanks", http.StatusOK)
		return
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// default method = GET
	if r.Method == "POST" {
		r.ParseForm()
		if r.FormValue("method") == "register" {
			err := data.Register(html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")),
				html.EscapeString(r.FormValue("safeword")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
			}

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
			resp, err := data.LoginVoter(currentElection.ID.Hex(), html.EscapeString(r.FormValue("username")),
				html.EscapeString(r.FormValue("password")), html.EscapeString(r.FormValue("safeword")), mdbClient)
			if err != nil {
				fmt.Printf(alert, err)
				http.Error(w, "Problem occurred", http.StatusTeapot)
				return
			}
			if !resp.Success {
				http.Error(w, "Unsuccessful login", http.StatusTeapot)
				return
			}
			if resp.HasVoted {
				http.Error(w, "Already voted", http.StatusTeapot)
				return
			}

			session.Values["id"] = resp.ID
			session.Save(r, w)

			tmpl := template.Must(template.ParseFiles("static/tmpl/election.html"))
			sd := "To be confirmed"
			ed := "To be confirmed"
			started := false
			if !currentElection.StartDate.IsZero() {
				if time.Now().After(currentElection.StartDate) {
					started = true
				}
				sd = currentElection.StartDate.Format("2006-01-02")
				ed = currentElection.EndDate.Format("2006-01-02")
			}
			data := struct {
				Started                       bool
				ID, StartDate, Title, EndDate string
				Options                       []string
			}{
				Started:   started,
				ID:        currentElection.ID.Hex(),
				StartDate: sd,
				Title:     currentElection.Title,
				EndDate:   ed,
				Options:   currentElection.Options,
			}
			tmpl.Execute(w, data)
		}
	} else {
		tmpl := template.Must(template.ParseFiles("static/tmpl/index.html"))
		tmpl.Execute(w, nil)
	}
}

func main() {
	// generate test certificate
	err := httpscerts.Check("cert.pem", "key.pem")
	if err != nil {
		err = httpscerts.Generate("cert.pem", "key.pem", "127.0.0.1:8080")
		if err != nil {
			fmt.Printf(alert, err)
			os.Exit(1)
		}
	}
	// end generate test certificate

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
	http.HandleFunc("/audit", auditHandler)
	http.HandleFunc("/submitvote", submitVoteHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/live", pubRecordHandler)

	// if err := http.ListenAndServe(":8080", nil); err != nil {
	// 	fmt.Printf(alert, fmt.Sprintf("failed to serve: %v", err))
	// 	os.Exit(1)
	// }
	http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil)
}
