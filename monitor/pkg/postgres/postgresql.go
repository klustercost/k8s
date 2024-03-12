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
// This function inserts the details of a pod into the database
func (pg *persistence_pg) InsertPod(pod_name, namespace string, podMisc *model.PodMisc, ownerRef *model.OwnerReferences, podUsage *model.PodConsumption) error {

	_, err := pg.db_connection.Exec("INSERT INTO klustercost.tbl_pods(time, namespace, app, service, pod, node, CPU, mem, shard) VALUES($1, $2, $3, NULLIF($4,''), $5, NULLIF($6,''), NULLIF($7,''),NULLIF($8,''), NULLIF($9,''))",
		podMisc.RecordTime, namespace, podMisc.AppLabel, "", pod_name, podMisc.NodeName, podUsage.CPU.Value, podUsage.Memory.Value, podMisc.Shard)
	if err != nil {
		fmt.Println("Error inserting pod details into the database:", err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED POD:", pod_name, namespace, "memory usage", podUsage.Memory, "CPU usage", podUsage.CPU, "owner", ownerRef.OwnerName, "node", podMisc.NodeName)
	return nil
}

// This function inserts the details of a node into the database
// price_per_hour to be added to the function argument and to the query once it is actually defined
func (pg *persistence_pg) InsertNode(node_name string, nodeMisc *model.NodeMisc) error {
	_, err := pg.db_connection.Exec("CALL add_node($1, $2, $3)",
		node_name, nodeMisc.Memory, nodeMisc.CPU)
	if err != nil {
		fmt.Println("Error inserting node details into the database:", err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED Node:", node_name, "memory", nodeMisc.Memory, "CPU", nodeMisc.CPU, "UID", nodeMisc.UID)
	return nil
}

// This function inserts the details of an owner into the database
// func InsertOwner(name string, namespace string, record_time time.Time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels string) error {
func (pg *persistence_pg) InsertOwner(name string, namespace string, allRef *model.AppOwnerReferences) error {

	_, err := pg.db_connection.Exec("INSERT INTO klustercost.tbl_owners(name, namespace, record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels) VALUES($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), NULLIF($10,''), NULLIF($11,''))",
		name, namespace, allRef.RecordTime, allRef.OwnVersion, allRef.OwnKind, allRef.OwnerUid, allRef.OwnerVersion, allRef.OwnKind, allRef.OwnerName, allRef.OwnerUid, allRef.Labels)
	if err != nil {
		fmt.Println("Error inserting owner details into the database:", err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED Owner:", name, namespace, "owner kind", allRef.OwnKind)
	return nil
}

// This function inserts the details of a service into the database
// func InsertService(name string, namespace string, own_uid, app_label, labels, selector string) error {
func (pg *persistence_pg) InsertService(name string, namespace string, svcRef *model.ServiceMisc) error {

	_, err := pg.db_connection.Exec("INSERT INTO klustercost.tbl_services(service_name, namespace, own_uid, app_label, labels, selector) VALUES($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''))",
		name, namespace, svcRef.UID, svcRef.AppLabel, svcRef.Labels, svcRef.Selector)
	if err != nil {
		fmt.Println("Error inserting service details into the database:", err)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		return err
	}
	fmt.Println("INSERTED Service:", name, namespace, "service selector", svcRef.Selector)
	return nil
}
