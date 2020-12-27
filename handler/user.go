package handler

import (
	dblayer "LookForYou/db"
	"LookForYou/util"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	pwd_salt = "#890"
)

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data, err := ioutil.ReadFile("./static/view/signup.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Failed to direct signup html, err:" + err.Error())
			return
		}
		w.Write(data)
		return
	}
	r.ParseForm()
	username := r.Form.Get("username")
	passwd := r.Form.Get("password")
	if len(username) < 3 || len(passwd) < 5 {
		w.Write([]byte("Invalid parameter"))
		return
	}
	enc_passwd := util.Sha1([]byte(passwd + pwd_salt))
	suc := dblayer.UserSignup(username, enc_passwd)
	if suc {
		w.Write([]byte("Sign Up success"))
	}
	w.Write([]byte("Sign Up fail"))
}
