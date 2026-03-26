package postgres

import (
	"database/sql"
	"fmt"
	"klustercost/monitor/pkg/env"
	"klustercost/monitor/pkg/model"
	"time"

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
		maxRetries := 30
		retryDelay := 2 * time.Second

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = db_connection.Ping()
			if err == nil {
				fmt.Println("Connected to PostgreSQL")
				break
			}
			if attempt == maxRetries {
				fmt.Println("Could not connect to PostgreSQL after", maxRetries, "attempts")
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
			fmt.Printf("PostgreSQL not ready, retrying in %v (attempt %d/%d)\n", retryDelay, attempt, maxRetries)
			time.Sleep(retryDelay)
			if retryDelay < 10*time.Second {
				retryDelay *= 2
			}
		}
		persistence_impl = &persistence_pg{db_connection}
	}

	return persistence_impl
}

func (pg *persistence_pg) Close() {
	persistence_impl.db_connection.Close()
}

// This function inserts the details of a pod into the database
// It calls the klustercost.register_pod_data stored procedure
func (pg *persistence_pg) InsertPod(pod_name, namespace, node string, podUsage *model.PodConsumption, appLabels *model.PodAppLabels, podResources *model.PodResources) error {
	_, err := pg.db_connection.Exec(
		"CALL klustercost.register_pod_data($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10,''), NULLIF($11,''), NULLIF($12,''), NULLIF($13,''), NULLIF($14,''), NULLIF($15,''))",
		pod_name, namespace, node,
		podUsage.CPU.Value, podUsage.Memory.Value,
		podResources.CPURequest, podResources.CPULimit, podResources.MemRequest, podResources.MemLimit,
		appLabels.Name, appLabels.Instance, appLabels.Version,
		appLabels.Component, appLabels.PartOf, appLabels.ManagedBy)
	if err != nil {
		return fmt.Errorf("error inserting pod details: %w", err)
	}
	fmt.Println("INSERTED POD:", pod_name, namespace, "memory usage", podUsage.Memory, "CPU usage", podUsage.CPU, "node", node)
	return nil
}

// This function inserts the details of a node into the database
// price_per_hour to be added to the function argument and to the query once it is actually defined
func (pg *persistence_pg) InsertNode(node_name string, nodeMisc *model.NodeMisc) error {
	_, err := pg.db_connection.Exec("CALL add_node($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), NULLIF($7,''), NULLIF($8,''))",
		node_name, nodeMisc.Memory, nodeMisc.CPU,
		nodeMisc.Labels, nodeMisc.InstanceType, nodeMisc.Region, nodeMisc.Zone, nodeMisc.OS)
	if err != nil {
		return fmt.Errorf("error inserting node details: %w", err)
	}
	fmt.Println("INSERTED Node:", node_name, "memory", nodeMisc.Memory, "CPU", nodeMisc.CPU, "labels", nodeMisc.Labels)
	return nil
}

// This function inserts the details of an owner into the database
// func InsertOwner(name string, namespace string, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels string) error {
func (pg *persistence_pg) InsertOwner(name string, namespace string, allRef *model.AppOwnerReferences) error {

	_, err := pg.db_connection.Exec("INSERT INTO klustercost.tbl_owners(name, namespace,  own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels) VALUES($1, $2, $3, $4, NULLIF($5,''), NULLIF($6,''), NULLIF($7,''),NULLIF($8,''), NULLIF($9,''), NULLIF($10,''))",
		name, namespace, allRef.OwnVersion, allRef.OwnKind, allRef.OwnerUid, allRef.OwnerVersion, allRef.OwnKind, allRef.OwnerName, allRef.OwnerUid, allRef.Labels)
	if err != nil {
		return fmt.Errorf("error inserting owner details: %w", err)
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
		return fmt.Errorf("error inserting service details: %w", err)
	}
	fmt.Println("INSERTED Service:", name, namespace, "service selector", svcRef.Selector)
	return nil
}
