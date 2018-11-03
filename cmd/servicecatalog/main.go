package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/carolynvs/cloudkinds-servicecatalog/pkg/servicecatalog"
)

func main() {
	fmt.Println("Service Catalog Provider reporting for duty! 🤖")
	fmt.Println("Listening on *:8080")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Received %s\n", r.URL.Path)
		defer r.Body.Close()
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			fmt.Fprintf(w, "%s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Printf("\t%v\n", string(payload))

		result, err := servicecatalog.DealWithIt(payload)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}
		fmt.Printf("\t%v\n", string(result))
		fmt.Fprintf(w, "%v", string(result))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
