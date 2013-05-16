package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"pisearch"
	"strconv"
	"time"
)

type Piserver struct {
	searcher *pisearch.Pisearch
}

func intmax(x, y int) int {
	if x > y {
		return x
	}
	return y
}

type jsonhandler func(*http.Request, map[string]interface{})

func (handler jsonhandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	results := make(map[string]interface{})
	err := req.ParseForm()
	if err != nil {
		results["status"] = "FAILED"
		results["error"] = "Bad form"
	} else {
		handler(req, results)
	}

	w.Header().Set("Content-Type", "text/javascript")
	results["elapsedTime"] = time.Now().Sub(startTime)
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		io.WriteString(w, "Fucked, can't even marshal the output for you.\n")
		return
	}
	if b != nil {
		io.WriteString(w, string(b))
	}
}

func (ps *Piserver) ServeDigits(req *http.Request, results map[string]interface{}) {
	results["status"] = "FAILED"
	startstr, has_start := req.Form["start"]
	countstr, has_count := req.Form["count"]
	if !has_start || !has_count {
		results["error"] = "Missing query parameters"
		return
	}
	start64, err := strconv.ParseInt(startstr[0], 10, 32)
	if err != nil {
		results["error"] = "Bad start position"
		return
	}
	start := int(start64)
	count, err := strconv.Atoi(countstr[0])
	if err != nil {
		results["error"] = "Bad count"
		return
	}
	results["status"] = "success"
	results["start"] = start
	results["count"] = count
	results["digits"] = ps.searcher.GetDigits(start, count)
}

func (ps *Piserver) ServeQuery(req *http.Request, results map[string]interface{}) {
	// results["status"] = ...
	// results["results"] = [ [result1], [result2], ... ]
	results["status"] = "OK"
	q, has_q := req.Form["q"]
	start, has_start := req.Form["start"]
	if !has_q {
		results["status"] = "FAILED"
		results["error"] = "Missing query"
		return
	}

	if len(q) > 20 {
		results["status"] = "FAILED"
		results["error"] = "Too many queries"
		return
	}

	start_pos := int(0)
	if has_start {
		sp, err := strconv.ParseInt(start[0], 10, 64)
		start_pos = int(sp)
		if err != nil {
			results["status"] = "FAILED"
			results["error"] = "Bad start position"
			return
		}
	}
	resarray := make([]map[string]interface{}, len(q))
	results["results"] = resarray
	startTime := time.Now()
	for idx, query := range q {
		m := make(map[string]interface{})
		m["searchKey"] = query
		m["start"] = start_pos
		if (start_pos > 0) { start_pos -= 1 }
		found, pos := ps.searcher.Search(start_pos, query)
		if found {
			digitBeforeStart := intmax(0, pos-20)
			m["status"] = "found"
			m["piPosition"] = pos+1 // 1 based indexing for humans
			m["digitsBefore"] = ps.searcher.GetDigits(digitBeforeStart, int(pos-digitBeforeStart))
			m["digitsAfter"] = ps.searcher.GetDigits(pos+len(query), 20)
		} else {
			m["status"] = "notfound"
		}
		endTime := time.Now()
		m["lookupTime"] = endTime.Sub(startTime)
		startTime = endTime
		resarray[idx] = m
	}
}

func main() {
	pifile := "/home/dga/public_html/pi/pi200"
	pisearch, err := pisearch.Open(pifile)
	if err != nil {
		log.Fatal("Could not open ", pifile, ": ", err)
	}
	server := &Piserver{pisearch}
	http.Handle("/piquery",
		jsonhandler(func(req *http.Request, respmap map[string]interface{}) {
			server.ServeQuery(req, respmap)
		}))
	http.Handle("/pidigits",
		jsonhandler(func(req *http.Request, respmap map[string]interface{}) {
			server.ServeDigits(req, respmap)
		}))

	werr := http.ListenAndServe(":1415", nil)
	if werr != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
