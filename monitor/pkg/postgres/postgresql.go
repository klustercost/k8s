package postgres

import (
	"database/sql"
	"fmt"
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

func NewPostgresql(user, password, dbname string) (*Postgresql, error) {
	p := &Postgresql{}
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)
	var err error
	p.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		klog.Fatal(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	return p, err
}

// This is a preliminary function to test the best way to insert a pod details the database
// For the moment it just inserts the namespace and name of the found pod
// Further DB structure to be defined
func (p *Postgresql) InsertPod(pod_name, namespace string, record_time time.Time, used_mem, used_cpu int64, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name string) error {

	_, err := p.DB.Exec("INSERT INTO klustercost.tbl_pods(pod_name, namespace, record_time, used_mem, used_cpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)	VALUES($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), $10, $11, $12)",
		pod_name, namespace, record_time, used_mem, used_cpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	return nil
}

func (p *Postgresql) Close() {
	p.DB.Close()
}

func (p *Postgresql) InsertNode(node_name string, creation_time time.Time, node_mem, node_cpu int64, node_uid string) error {

	_, err := p.DB.Exec("INSERT INTO klustercost.tbl_nodes(node_name, node_creation_time, node_mem, node_cpu, node_uid) VALUES($1, $2, $3, $4, $5)",
		node_name, creation_time, node_mem, node_cpu, node_uid)
	if err != nil {
		klog.Error(err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	return nil
}
