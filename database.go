package main

import (
"bytes"
"fmt"
"github.com/jackc/pgx"
"io"
"io/ioutil"
"log"
"net/http"
"net/url"
"strings"
)

type requests struct {
	Id int64
	Method string
	Url string
	Headers string
	Body string
}

type AllRequests []requests

type writer struct {
	bytes.Buffer
}

func (b *writer) Close() error {
	b.Buffer.Reset()
	return nil
}

func InitDatabase() *pgx.ConnPool{
	config := pgx.ConnPoolConfig {
			ConnConfig:     pgx.ConnConfig{
			Host:                 "localhost",
			Port:                 5432,
			Database:             "proxy",
			User:                 "tpark",
			Password:             "TP2021",
		},
		MaxConnections: 100,
	}
	connPool, err := pgx.NewConnPool(config)
	if err != nil {
		log.Fatal(err)
	}
	return connPool
}

func LogRequest(r *http.Request, db *pgx.ConnPool) error {
	body, err := ioutil.ReadAll(r.Body)
	var id int
	if err != nil {
		return err
	}
	headers := ""
	for key, val := range r.Header {
		headers += key + ": " + val[0] + "\n"
	}
	err = db.QueryRow("INSERT INTO req VALUES(default, $1, $2, $3, $4) RETURNING id",
		r.URL.String(), r.Method, string(body), headers).Scan(&id)
	return err
}

func GetRequest(id int, db *pgx.ConnPool) http.Request {
	var result http.Request
	var request requests
	err := db.QueryRow("SELECT * FROM req WHERE id = $1", id).Scan(&request.Id, &request.Url, &request.Method, &request.Body, &request.Headers)
	if err != nil {
		fmt.Println(err)
		return http.Request{}
	}
	result.Method = request.Method
	result.URL, err = url.Parse(request.Url)
	if err != nil {
		return http.Request{}
	}
	var bodyWriter io.ReadWriteCloser
	bodyWriter = &writer{}
	_, err = bodyWriter.Write([]byte(request.Body))
	if err != nil {
		return http.Request{}
	}
	result.Body = bodyWriter
	headMap := make(map[string][]string)
	for _, val := range strings.Split(request.Headers, "\n") {
		if val != "" {
			buf := strings.Split(val, ":")
			headMap[buf[0]] = []string{buf[1]}
		}
	}
	result.Header = headMap
	return result
}

func GetAllRequests(db *pgx.ConnPool) AllRequests{
	var allReq AllRequests
	rows, err := db.Query("SELECT * FROM req")
	if err != nil {
		return allReq
	}
	defer rows.Close()
	for rows.Next() {
		request := requests{}
		err := rows.Scan(&request.Id, &request.Url, &request.Method, &request.Body, &request.Headers)
		if err != nil {
			return allReq
		}
		allReq = append(allReq, request)
	}
	return allReq
}

func GetRequestInfo(db *pgx.ConnPool, id int) *requests{
	row := db.QueryRow("SELECT * FROM req WHERE id=$1", id)
	request := &requests{}
	err := row.Scan(&request.Id, &request.Url, &request.Method, &request.Body, &request.Headers)
	if err != nil {
		return nil
	}
	return request
}
