package main

import (
	"chat/trace"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
)

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}

	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func getEnv(varName string) string {
	// varName := "OAUTH_CLIENT_ID"
	// varName := "OAUTH_CLIENT_SECRET"
	value := os.Getenv(varName)
	if value == "" {
		panic(fmt.Sprintf("%s is not set", varName))
	}
	return value
}

func main() {
	var addr = flag.String("addr", ":8080", "The address of the application.")
	flag.Parse()

	gomniauth.SetSecurityKey("MyRandomKey")
	gomniauth.WithProviders(
		google.New(
			getEnv("OAUTH_CLIENT_ID"),
			getEnv("OAUTH_CLIENT_SECRET"),
			"http://localhost:8080/auth/callback/google",
		),
	)

	r := newRoom(UseGravatar)
	r.tracer = trace.New(os.Stdout)
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)

	go r.run()

	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
