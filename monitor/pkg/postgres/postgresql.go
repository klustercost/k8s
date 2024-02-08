package postgres

import (
	"database/sql"
	"fmt"
	"klustercost/monitor/pkg/env"
	"time"

	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

var db_connection *sql.DB = nil

func init() {
	env := env.NewConfiguration()
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", env.PgDbUser, env.PgDbPass, env.PgDbName, env.PgDbHost, env.PgDbPort)
	var err error
	db_connection, err = sql.Open("postgres", connectionString)
	if err != nil {
		klog.Fatal(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

}

// Closes the connection to the DB.
func Close() {
	if db_connection != nil {
		db_connection.Close()
	}
}

// This function inserts the details of a pod into the database
func InsertPod(pod_name, namespace string, record_time time.Time, used_mem, used_cpu int64, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name string) error {
	_, err := db_connection.Exec("INSERT INTO klustercost.tbl_pods(pod_name, namespace, record_time, used_mem, used_cpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)	VALUES($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), $10, $11, $12)",
		pod_name, namespace, record_time, used_mem, used_cpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED POD:", pod_name, namespace, record_time, used_mem, used_cpu, owner_name, node_name)
	return nil
}

// This function inserts the details of a node into the database
// price_per_hour to be added to the function argument and to the query once it is actually defined
func InsertNode(node_name string, node_mem, node_cpu int64, node_uid, labels string) error {
	_, err := db_connection.Exec("INSERT INTO klustercost.tbl_nodes(node_name, node_mem, node_cpu, node_uid, labels, price_per_hour) VALUES($1, $2, $3, $4, $5, NULLIF($6,''))",
		node_name, node_mem, node_cpu, node_uid, labels, "")
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED Node:", node_name, node_mem, node_cpu, node_uid)
	return nil
}

// This function inserts the details of an owner into the database
func InsertOwner(name string, namespace string, record_time time.Time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels string) error {
	_, err := db_connection.Exec("INSERT INTO klustercost.tbl_owners(name, namespace, record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels) VALUES($1, $2, $3, $4, $5, $6, NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), NULLIF($10,''), NULLIF($11,''))",
		name, namespace, record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED Owner:", name, namespace, record_time, own_kind, own_uid)
	return nil
}
