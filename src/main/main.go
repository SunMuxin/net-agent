package main

import (
	"fmt"
	_ "github.com/icattlecoder/godaemon"
	"log"
	"net/http"
	"process"
)

func WindowsTopview(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	var walk = process.WalkerWindows()
	topview, err := walk.Walk()
	if err != nil {
		_, _ = fmt.Fprintf(w, err.Error())
		return
	}

	_, _ = fmt.Fprintf(w, topview.GetMessage(true))
}

func LinuxTopview(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	var walk = process.WalkerWindows()
	topview, err := walk.Walk()
	if err != nil {
		_, _ = fmt.Fprintf(w, err.Error())
		return
	}

	_, _ = fmt.Fprintf(w, topview.GetMessage(true))
}

func main() {
	http.HandleFunc("/net/topview", LinuxTopview)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
