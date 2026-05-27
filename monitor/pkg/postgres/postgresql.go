package postgres

import (
	"database/sql"
	"fmt"
	"klustercost/monitor/pkg/env"
	"klustercost/monitor/pkg/model"

	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

type persistence_pg struct {
	db_connection *sql.DB
}

var persistence_impl *persistence_pg = nil

// Closes the connection to the DB.
func ClosePersistInterface() {
	if persistence_impl != nil {
		persistence_impl.Close()
	}
}
func GetPersistInterface() interface{} {
	if persistence_impl == nil {
		env := env.NewConfiguration()
		connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", env.PgDbUser, env.PgDbPass, env.PgDbName, env.PgDbHost, env.PgDbPort)
		db_connection, err := sql.Open("postgres", connectionString)
		if err != nil {
			fmt.Println("Error opening the DB connection:", err)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		persistence_impl = &persistence_pg{db_connection}
	}

	return persistence_impl
}

func (pg *persistence_pg) Close() {
	persistence_impl.db_connection.Close()
}

// This function inserts the details of a pod into the database
// It calls the klustercost.register_pod_data_json stored procedure
func (pg *persistence_pg) InsertPodJson(pod_json string) error {
	_, err := pg.db_connection.Exec("CALL klustercost.register_pod_json($1)", pod_json)
	if err != nil {
		fmt.Println("Error inserting pod details into the database:", err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	return nil
}

// This function inserts the details of a node into the database
// price_per_hour to be added to the function argument and to the query once it is actually defined
func (pg *persistence_pg) InsertNode(node_name string, nodeMisc *model.NodeMisc) error {
	_, err := pg.db_connection.Exec("CALL add_node($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), NULLIF($7,''), NULLIF($8,''))",
		node_name, nodeMisc.Memory, nodeMisc.CPU,
		nodeMisc.Labels, nodeMisc.InstanceType, nodeMisc.Region, nodeMisc.Zone, nodeMisc.OS)
	if err != nil {
		fmt.Println("Error inserting node details into the database:", err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED Node:", node_name, "memory", nodeMisc.Memory, "CPU", nodeMisc.CPU, "labels", nodeMisc.Labels)
	return nil
}
