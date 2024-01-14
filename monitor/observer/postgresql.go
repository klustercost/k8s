package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

// DB connection parameters
// the best way to handle credentials is TBD
// At the moment they are hardcoded and the DB is hosted locally

type Postgresql struct {
	DB *sql.DB
}

// Init the connection to the postgreds database
func (p *Postgresql) Initialize(user, password, dbname string) error {
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error
	p.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer p.DB.Close()
	return err
}

// This is a preliminary function to test the best way to insert a pod details the database
// For the moment it just inserts the namespace and name of the found pod
// Further DB structure to be defined
func (p *Postgresql) InsertPod(ns, name string, currmemusg int64, timestamp time.Time) {

	_, err := p.DB.Exec("INSERT INTO ns_pod.ns_pod(namespace, name, currmemusg, timestamp) VALUES($1, $2, $3, $4)", ns, name, currmemusg, timestamp)
	if err != nil {
		klog.Info(err)
		return
	}
}
