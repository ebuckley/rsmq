package main

import (
	_ "embed"
	"fmt"
	"github.com/ebuckley/rsmq/q"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
)

func main() {

	useOS := len(os.Args) > 1 && os.Args[1] == "live"
	//
	nr := mux.NewRouter()

	// handle the index
	nr.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		queue, err := q.New(req.Context(), q.Options{})
		if err != nil {
			http.Error(res, err.Error(), 500)
		}
		queues, err := queue.ListQueues(req.Context())
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}

		tpl := mustTemplate(useOS, "templates/ConnectionList.html")
		err = tpl.Execute(res, struct {
			Qcount int
			Queues []string
		}{
			Qcount: len(queues),
			Queues: queues,
		})
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}
	})

	// handle a q
	nr.HandleFunc("/q/{qname}", func(w http.ResponseWriter, r *http.Request) {
		vs := mux.Vars(r)
		qname, ok := vs["qname"]
		if !ok {
			http.Error(w, "Not found", 404)
			return
		}
		queue, err := q.New(r.Context(), q.Options{})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if r.Method == http.MethodDelete {
			err := queue.DeleteQueue(r.Context(), q.DeleteQueueRequestOptions{QName: qname})
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(200)
			fmt.Fprint(w, `
                <div class='pb-8 pt-8 bg-red-300 text-center text-red-800 font-xl font-bold'>
                    DELETED `+qname+`
                </div>
            `)
			return
		}

		type ConnectionDetails struct {
			Name  string
			Attrs *q.QueueAttributes
		}

		attrs, err := queue.GetQueueAttributes(r.Context(), q.GetQueueAttributesOptions{QName: qname})
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		tpl := mustTemplate(useOS, "templates/ConnectionDetail.html")
		tpl.Execute(w, ConnectionDetails{
			Name:  qname,
			Attrs: attrs,
		})
	})
	// handle a q edit
	nr.HandleFunc("/q/{qname}/edit", func(w http.ResponseWriter, r *http.Request) {
		vs := mux.Vars(r)
		qname, ok := vs["qname"]
		if !ok {
			http.Error(w, "Not found", 404)
			return
		}
		queue, err := q.New(r.Context(), q.Options{})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if r.Method == http.MethodPost {
			//qUpdate := q.SendMessageRequestOptions{
			//    QName:   "",
			//    Delay:   0,
			//    Message: "",
			//}
			//TODO handle update mode
			//err := queue.DeleteQueue(r.Context(), q.DeleteQueueRequestOptions{QName: qname})
			//if err != nil {
			//	http.Error(w, err.Error(), 500)
			//	return
			//}
			//w.WriteHeader(200)
			//// TODO
			//fmt.Fprint(w, `
			//    <div class='pb-8 pt-8 bg-red-300 text-center text-red-800 font-xl font-bold'>
			//        Updated `+qname+`
			//    </div>
			//`)
			return
		}

		type ConnectionDetails struct {
			Name  string
			Attrs *q.QueueAttributes
		}

		attrs, err := queue.GetQueueAttributes(r.Context(), q.GetQueueAttributesOptions{QName: qname})
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		tpl := mustTemplate(useOS, "templates/ConnectionEdit.html")
		tpl.Execute(w, ConnectionDetails{
			Name:  qname,
			Attrs: attrs,
		})
	})
	nr.PathPrefix("/").Handler(http.FileServer(getFileSystem(useOS, embededFiles)))

	//addUserRoutes(nr.PathPrefix("/user/").Subrouter())
	// print out registered routes
	//printrouter(nr)

	log.Println("STARTING ON :8989")
	err := http.ListenAndServe(":8989", nr)
	if err != nil {
		log.Fatalln(err)
	}
}
