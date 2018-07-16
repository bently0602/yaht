package main

import (
	"fmt"
	"io/ioutil"
	"github.com/spf13/viper"
	"net/http"
	"path"
	"strings"
	"github.com/sec51/twofactor"
	"crypto"
	b64 "encoding/base64"
)

func ServePage(w http.ResponseWriter, page string) {
    exPath := GetExePath()
	file, err := ioutil.ReadFile(path.Join(exPath, page))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Fprint(w, string(file))	
	}
}

// https://stackoverflow.com/questions/28793619/golang-what-to-use-http-servefile-or-http-fileserver
func StartAdminWebApp() {
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		ServePage(w, "static/index.html")
	})

	http.HandleFunc("/generatetotp", func (w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := strings.Join(r.Form["name"], "")
		otp, err := twofactor.NewTOTP(name, viper.GetString("instanceName"), crypto.SHA1, 8)	
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{\"success\": false}")			
			return
		}
		qrBytes, err := otp.QR()
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{\"success\": false}")			
			return
		}
		qrBase64 := b64.StdEncoding.EncodeToString(qrBytes)
		secret, err := otp.ToBytes()
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{\"success\": false}")			
			return
		}		
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		
		fmt.Fprint(w, fmt.Sprintf("{\"success\":true, \"secret\": \"%s\", \"qrCode\": \"%s\"}", b64.StdEncoding.EncodeToString(secret), qrBase64))
	})


	http.HandleFunc("/load", func (w http.ResponseWriter, r *http.Request) {
		file, err := ioutil.ReadFile(viper.ConfigFileUsed())
		if err != nil {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, err)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(file))
		}	
	})

	http.HandleFunc("/save", func (w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		configSource := strings.Join(r.Form["config_source"], "")
		err := ioutil.WriteFile(viper.ConfigFileUsed(), []byte(configSource), 0666)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{\"success\": false}")
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{\"success\": true}")
		}
	})

	fs := http.FileServer(http.Dir(path.Join(GetExePath(), "/static/")))
	http.Handle("/static/", http.StripPrefix("/static", fs))

	http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("adminWeb.port")), nil)
}