package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

var db_connection *sql.DB = nil

func init() {
	user := os.Getenv("PG_DB_USER")
	if user == "" {
		user = "postgres"
		klog.Info("PG_DB_USER not set, using default value of postgres")
	}

	password := os.Getenv("PG_DB_PASS")
	if password == "" {
		password = "admin"
		klog.Info("PG_DB_PASS not set, using default value of admin")
	}

	dbname := os.Getenv("PG_DB_NAME")
	if dbname == "" {
		dbname = "klustercost"
		klog.Info("PG_DB_NAME not set, using default value of klustercost")
	}

	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error

	db_connection, err = sql.Open("postgres", connectionString)

	if err != nil {
		klog.Fatal(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}

func Close() {
	if db_connection != nil {
		db_connection.Close()
	}
}

// This is a preliminary function to test the best way to insert a pod details the database
// For the moment it just inserts the namespace and name of the found pod
// Further DB structure to be defined
func InsertPod(pod_name, namespace string, record_time time.Time, used_mem, used_cpu int64, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name string) error {

	_, err := db_connection.Exec("INSERT INTO klustercost.tbl_pods(pod_name, namespace, record_time, used_mem, used_cpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)	VALUES($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), $10, $11, $12)",
		pod_name, namespace, record_time, used_mem, used_cpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	return nil
}

func InsertNode(node_name string, creation_time time.Time, node_mem, node_cpu int64, node_uid string) error {

	_, err := db_connection.Exec("INSERT INTO klustercost.tbl_nodes(node_name, node_creation_time, node_mem, node_cpu, node_uid) VALUES($1, $2, $3, $4, $5)",
		node_name, creation_time, node_mem, node_cpu, node_uid)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	return nil
}

func InsertOwners(name string, namespace string, record_time time.Time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels string) error {
	_, err := db_connection.Exec("INSERT INTO klustercost.tbl_owners(name, namespace, record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels) VALUES($1, $2, $3, $4, $5, $6, NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), NULLIF($10,''), NULLIF($11,''))",
		name, namespace, record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	return nil
}
