package main

import (
	"fmt"
	"flag"
	"github.com/gliderlabs/ssh"
	"io"
	"log"
	"time"
	"os"	
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
	"context"
	"path"
	"strings"
	"github.com/sec51/twofactor"
	b64 "encoding/base64"
)

func generateServer() (*ssh.Server) {
	server := &ssh.Server{
		Addr: fmt.Sprintf(":%d", viper.GetInt("ssh.port")),
		MaxTimeout: time.Duration(viper.GetInt64("ssh.deadTimeoutMinutes") * int64(time.Minute)),
		IdleTimeout: time.Duration(viper.GetInt64("ssh.idleTimeoutMinutes") * int64(time.Minute)),
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			return false
		},
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			addr := fmt.Sprintf("%s:%d", destinationHost, destinationPort)
			userPrefix := "users." + ctx.User()
			if viper.IsSet(userPrefix + ".routes") == false {
				return false
			}

			t := viper.GetStringSlice(userPrefix + ".routes")
			for _, v := range t {
				if v == addr {
					return true
				}
			}

			return false
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			userPrefix := "users." + ctx.User()
			if viper.IsSet(userPrefix + ".authType") == false {
				return false
			}

			if viper.GetString(userPrefix + ".authType") == "password" {
				if viper.IsSet(userPrefix + ".value") == false {
					return false
				}

				if viper.GetString(userPrefix + ".value") == password {
					return true
				}
			}

			if viper.GetString(userPrefix + ".authType") == "password+totp" {
				if viper.IsSet(userPrefix + ".value") == false {
					return false
				}
				if viper.IsSet(userPrefix + ".totpToken") == false {
					return false
				}

				s := strings.SplitN(password, " ", 2)
				if len(s) != 2 {
					return false
				}
				sshPassword, sshTOTP := s[0], s[1]

				if viper.GetString(userPrefix + ".value") != sshPassword {
					return false
				}

				totpToken := viper.GetString(userPrefix + ".totpToken")
				totpTokenBytes, err := b64.StdEncoding.DecodeString(totpToken)
				if err != nil {
					return false
				}
				otp, err := twofactor.TOTPFromBytes(totpTokenBytes, viper.GetString("instanceName"))
				if err != nil {
					return false
				}				
				err = otp.Validate(sshTOTP)
				if err != nil {
					return false
				}

				return true
			}

			return false
		},
		Handler: func(s ssh.Session) {
			io.WriteString(s, fmt.Sprintf("Welcome %s\n", s.User()))
			i := 0
			for {
				i += 1
				select {
				case <-time.After(time.Duration(viper.GetInt64("ssh.heartbeatSeconds") * int64(time.Second))):
					continue
				case <-s.Context().Done():
					log.Println("Connection closed for", s.User())
					return
				}
			}		
		},
	}

	// * load host key
	exPath := GetExePath()
	privateKeyPath := path.Join(exPath, "private.pem")	
	signer, err := loadPrivatePEMKeyAsSSHSigner(privateKeyPath)
	if err != nil {
		log.Fatalln(fmt.Errorf("Fatal error reading host key: %s \n", err))
	}
	server.AddHostKey(signer)

	return server
}

func main() {
	exPath := GetExePath()

	var configFilePath = flag.String(
		"config", path.Join(exPath, "config.json"), "Path to config file",
	)
	flag.Parse()

	// check if host key exists, if not create
	privateKeyPath := path.Join(exPath, "private.pem")
	publicKeyPath := path.Join(exPath, "public.pem")
	if _, err := os.Stat(privateKeyPath); err != nil {
		if os.IsNotExist(err) {
			log.Println("Generating host key...")

			key, errB := generateRSAKey(2048)
			if errB != nil {
				log.Fatalln(fmt.Errorf("Fatal error generating host key: %s \n", errB))
			}

			publicKey := key.PublicKey
			savePrivatePEMKey(privateKeyPath, key)
			savePublicPEMKey(publicKeyPath, publicKey)
		} else {
			log.Fatalln(fmt.Errorf("Fatal error opening host key: %s \n", err))
		}
	}

	// open config file and set defaults for minimal runnable
	viper.SetConfigType("json")
	viper.SetConfigFile(*configFilePath)
	// read in config file and check for errors
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	viper.SetDefault("ssh.port", 2222)
	viper.SetDefault("ssh.idleTimeoutMinutes", 30)
	viper.SetDefault("ssh.deadTimeoutMinutes", 60)
	viper.SetDefault("ssh.heartbeatSeconds", 5)
	viper.SetDefault("reloadOnConfigChange", false)
	viper.SetDefault("instanceName", "yahtyahtyaht")

	log.Println("Using config =>", viper.ConfigFileUsed())

	// start up admin web server
	log.Println(fmt.Sprintf("Starting admin web server on :%d...", viper.GetInt("adminWeb.port")))
	go StartAdminWebApp()
	log.Println("DONE!")

	// generate server and watch for breaking config file changes
	log.Print("Generating SSH server...")
	server := generateServer()
	log.Println("DONE!")

	// Needs to be before...
	// stopped null pointer error for me
	// https://github.com/spf13/viper/issues/175
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
		if viper.GetBool("shutdownOnConfigChange") == true {
			server.Close()
			server.Shutdown(context.Background())
		}
	})	
	viper.WatchConfig()

	// start ssh server
	log.Println(fmt.Sprintf("Starting SSH server on :%d", viper.GetInt("ssh.port")))
	log.Println(server.ListenAndServe())
}
