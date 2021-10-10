package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func NewServer(port string, db *pgx.ConnPool) *http.Server {
	return &http.Server{
		Addr:         port,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				checkPattern := `^/scan/[0-9]+$`
				requestsPattern := `^/requests$`
				requestRepeatPattern := `^/repeat/[0-9]+$`
				requestPattern := `^/request/[0-9]+$`
				if match, _ := regexp.Match(requestsPattern, []byte(r.URL.String())); match {
					RequestList(w, r, db)
				} else if match, _ := regexp.Match(requestRepeatPattern, []byte(r.URL.String())); match {
					RepeatRequest(w, r, db)
				} else if match, _ := regexp.Match(checkPattern, []byte(r.URL.String())); match {
					CheckWithParamMiner(w, r, db)
				} else if match, _ := regexp.Match(requestPattern, []byte(r.URL.String())); match {
					RequestInfo(w, r, db)
				} else {
					handleHTTP(w, r, db)
				}
			}
		}),
	}
}

func handleHTTP(w http.ResponseWriter, r *http.Request,  db *pgx.ConnPool) {
	var resp *http.Response
	var err error
	err = LogRequest(r, db)
	if err != nil {
		return
	}
	switch r.Method {
	case "GET":
		resp, err = http.DefaultTransport.RoundTrip(r)
	case "POST":
		resp, err = http.Post(r.URL.String(), r.Header.Get("Content-Type"), r.Body)
	default:
		resp, err = http.Get(r.URL.String())
	}

	if err != nil {
		return
	}
	defer resp.Body.Close()
	for mime, val := range resp.Header {
		if mime == "Proxy-Connection" {
			continue
		}
		w.Header().Set(mime, val[0])
	}
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type")+"; charset=utf8")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func RequestList(w http.ResponseWriter, _ *http.Request, db *pgx.ConnPool) {
	result := GetAllRequests(db)
	answer, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(answer)
}

func RequestInfo(w http.ResponseWriter, r *http.Request, db *pgx.ConnPool) {
	buffer := strings.Split(r.URL.String(), "/")
	id, err := strconv.Atoi(buffer[2])
	if err != nil {
		return
	}
	request := GetRequestInfo(db, id)
	answer, err := json.Marshal(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(answer)
}

func RepeatRequest(w http.ResponseWriter, r *http.Request, db *pgx.ConnPool) {
	buffer := strings.Split(r.URL.String(), "/")
	id, err := strconv.Atoi(buffer[2])
	if err != nil {
		return
	}
	request := GetRequest(id, db)
	r = &request
	http.Redirect(w, &request, request.URL.String(), 301)
	return
}

func CheckWithParamMiner(w http.ResponseWriter, r *http.Request, db *pgx.ConnPool)  {
	flag := false
	buffer := strings.Split(r.URL.String(), "/")
	id, err := strconv.Atoi(buffer[2])
	if err != nil {
		return
	}
	request := GetRequest(id, db)

	if request.Method == "" {
		fmt.Println("request doesn't exist")
		return
	}
	for _, val := range GetParams() {
		randomString := RandStringRunes()
		request.URL.RawQuery = val+"="+randomString
		resp, err := http.DefaultTransport.RoundTrip(&request)
		if err != nil {
			fmt.Println("error with round trip")
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("error with Read all")
			return
		}
		if strings.Contains(string(body), randomString) {
			w.Write([]byte(val + "-found hidden GET params\n"))
			flag = true
		}
	}
	if flag == false {
		w.Write([]byte("No hidden GET params\n"))
	}
}
