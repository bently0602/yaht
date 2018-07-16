<img style="width:240px;" src="web/yaht.png" alt="yaht"/>

## What is Yaht

Yaht is a small utility to serve only ssh proxy requests. It is configurable to only allow specific ports to be proxied and allows users to use password and password + one time authentication (google authenticator, etc...).
Its a self contained program that is compiles across different platforms since it's written using Go. There is also a web configuration front end that allows someone to modify the configuration file over a web browser.

## How to Start

```

yaht (-config config.json)

```

## Setup

At startup, if a pem key pair is not in the exe's directory it will create one. If a config file is not passed as a parameter, it will try to load one in the exe's directory.

To setup users password and password+totp authentication is available. To get a token for the configuration file and a image to scan for google authenticator use the web configuration front end, access via localhost:8080 ( or whatever port adminweb.port is configurated to in config.json).

```
"test": {
	"authType": "password",
	"value": "password",
	"routes": [
		"127.0.0.1:8000"
	]
},
"testtotp": {
	"authType": "password+totp",
	"value": "password",
	"totpToken": "jAAAAAAAAADVBBYooYhRZSmvSsdajT9UxbtzfhWt5t3cnteHoeHJ9fZKmeKqlRj6CMK26iMHQPXVuqm/MTuw/4OUJNZKanJNZZQFz6Ri17YqqVcZ5Ja2Iwi7fVIxPyw41NcuIGS0k9p40BZchR9bzxz9X58G28m7hN0xD+R3/NTHbhyEOyFnVBeNuuQ=",
	"routes": [
		"127.0.0.1:8000"
	]
}
```

## Setup over Web

A web front end is available that allows you to generate totp tokens and barcodes for scanning in google authenticator. It allows you to edit the configuration file as well. There is no authentication on it so make sure the proper ports are blocked. By default, the web frontend is started on port 8080.

## Using

Use ssh to login to the server like normal, passing in local port forwarding options:

```
ssh -L 8888:127.0.0.1:8000 testtotp@127.0.0.1 -p 2222
```

When it prompts for a password:
1. Use just your password if authType = password for user
2. if authType = password+totp, use the password and token with a space between them.
	for example use testtotp for above, "password 12345678"
	note there is no space between the totp segemnts so 1234 5678 would be 12345678

## Testing Out

Start up a python web server in a terminal (default port for the following command is 8000):

```
python -m http.server
```

Then start yaht proxy.

Then run the following SSH command:

```
ssh -L 8888:127.0.0.1:8000 testtotp@127.0.0.1 -p 2222
```
