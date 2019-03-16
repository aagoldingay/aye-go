package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	utils "github.com/aagoldingay/aye-go/utilities"
	"github.com/globalsign/mgo"
)

const alert = "[ALERT] : %v\n"
const usernameTMPL = "aye-go"

func readConfig() ([]string, error) {
	// initial read
	b, err := ioutil.ReadFile("config.txt")
	if err != nil { // file not found
		fmt.Printf(alert, err)
		data, err := writeConfig()
		if err != nil {
			return []string{}, err
		}
		return data, nil
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

func main() {
	_, err := mgo.Dial("localhost:27017")
	if err != nil && err.Error() == "no reachable servers" {
		fmt.Printf(alert, "mongodb not installed")
		os.Exit(1)
	}

	utils.Setup(-1)
	// db := sess.DB("ayedb")

	// try reading a config.txt, if not exist, create one + db (if not exists.....)
	// if config.txt
	// read, login
	// else
	// db.UpsertUser(mgo.User{Username: })
	// read / update config.txt

	data, err := readConfig()
	addNewUser := false
	if err != nil {
		if err.Error() == "new user" {
			addNewUser = true
		}
		fmt.Printf(alert, err)
		os.Exit(1)
	}
	if len(data) != 2 {
		fmt.Printf(alert, "login data not as expected")
		os.Exit(1)
	}

	// successful config file
	if addNewUser {
		// db.UpsertUser(mgo.User{...})
	}
	// sess.Login(...)

	fmt.Println(data[0])
	fmt.Println(data[1])

	//fmt.Printf(alert, db.Name)
	//fmt.Println("aye-go")
}
