package main

import (
	"crypto/tls"
	"danielgarcia.net/snippetbox/pkg/models/mysql"
	"database/sql"
	"flag"
	"github.com/golangcollege/sessions"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/subosito/gotenv"
	_ "github.com/go-sql-driver/mysql"
)


func init() {
	gotenv.Load()
}

type contextKey string

const contextKeyIsAuthenticated = contextKey("isAuthenticated")

//application struct holding all dependencies for accessibility across the application currently only our custom loggers
type application struct {
	errorLog      *log.Logger
	infoLog       *log.Logger
	session       *sessions.Session
	snippets      *mysql.SnippetModel
	templateCache map[string]*template.Template
	users 		  *mysql.UserModel
}

func main() {
	dsn := flag.String("dsn", os.Getenv("DBNAME")+ ":" + os.Getenv("DBPASSWORD")+ "@" +
		"tcp(snippetbox1.chsxhrymwq6e.us-west-1.rds.amazonaws.com:3306)/snippetbox?parseTime=true", "MySQL data source name")
	addr := flag.String("addr", ":243", "HTTPS network address")
	secret := flag.String("secret", os.Getenv("SECRET_KEY"), "Secret key")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache("./ui/html/")
	if err != nil {
		errorLog.Fatal(err)
	}
	session := sessions.New([]byte(*secret))
	session.Lifetime = 12 * time.Hour

	// And add the session manager to our application dependencies.
	app := &application{
		errorLog:      errorLog,
		infoLog:       infoLog,
		session:       session,
		snippets:      &mysql.SnippetModel{DB: db},
		templateCache: templateCache,
		users: 		   &mysql.UserModel{DB: db},
	}

	// Initialize a tls.Config struct to hold the non-default TLS settings we want
	// the server to use.
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	// Set the server's TLSConfig field to use the tlsConfig variable we just
	// created.
	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		// Add Idle, Read and Write timeouts to the server.
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}



	infoLog.Printf("Starting server on %s", *addr)
	//serving file using custom server struct with
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
