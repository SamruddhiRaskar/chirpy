package main

import (
	"net/http"
)

// This is custom handler we can write this as per our logic. and this custom handler is one type of handle and second is fileserver
// Here we have created healthzHandler function in which we have pass parameter w is for write a response to the client or browser
// r is used for accepting the request from the http or client
// Then we have used various methods to print for get the content-type ,status and plain text msg.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	// this creates router which dont have routes yet thats why it is giving 404 not found

	fileserver := http.FileServer(http.Dir("."))
	// here dir(.) says that look at the current directory means whats in folder

	appHandler := http.StripPrefix("/app", fileserver)
	//this is strippredix method to remove prefix from dir

	// here "/" means it is asking for root path means root directory is index.html
	// if it use /style.css or /script.js then it is asking for css file and js file but using html
	//we can access both css and js due to linking them in html
	//mux.Handle("/", fileserver)
	mux.Handle("/app/", appHandler)

	mux.HandleFunc("/healthz", healthzHandler)
	// Now here we have called customHandler func

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
	//here it start the server

}
