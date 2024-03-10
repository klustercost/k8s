package env

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

// EnvVars is a struct that holds the env variables
type EnvVars struct {
	ResyncTime        int
	ControllerWorkers int
	PgDbUser          string
	PgDbPass          string
	PgDbName          string
	PgDbHost          string
	PgDbPort          string
	PrometheusServer  string
}

// InitEnvVars reads the env file and sets the env variables
// If the env file is not found, it uses the default values
func NewConfiguration() *EnvVars {
	//Open the env file and read the values
	file, err := os.Open("../config/env")
	logger := klog.Background()

	if err != nil {
		logger.Info("No env file found, using default values")
	}
	defer file.Close()

	//Read the file line by line and set the env variables
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}

	//Defualt values for the env variables
	result := &EnvVars{600, 2, "postgres", "admin", "klustercost", "localhost", "5432", "http://127.0.0.1:8080"}

	resync_time, err := strconv.Atoi(os.Getenv("RESYNC_TIME"))
	if err == nil {
		result.ResyncTime = resync_time
	} else {
		logger.Info("RESYNC_TIME not set, using default value of 600s")
	}

	controller_workers, err := strconv.Atoi(os.Getenv("CONTROLLER_WORKERS"))
	if err == nil {
		result.ControllerWorkers = controller_workers
	} else {
		logger.Info("CONTROLLER_WORKERS not set, using default value of 2")
	}

	pg_db_user := os.Getenv("PG_DB_USER")
	if pg_db_user != "" {
		result.PgDbUser = pg_db_user
	} else {
		logger.Info("PG_DB_USER not set, using default value")
	}

	pg_db_pass := os.Getenv("PG_DB_PASS")
	if pg_db_pass != "" {
		result.PgDbPass = pg_db_pass
	} else {
		logger.Info("PG_DB_PASS not set, using default value")
	}

	pg_db_name := os.Getenv("PG_DB_NAME")
	if pg_db_name != "" {
		result.PgDbName = pg_db_name
	} else {
		logger.Info("PG_DB_NAME not set, using default value")
	}

	pg_db_port := os.Getenv("PG_DB_PORT")
	if pg_db_port != "" {
		result.PgDbPort = pg_db_port
	} else {
		logger.Info("PG_DB_PORT not set, using default value 5432")
	}

	pg_db_host := os.Getenv("PG_DB_HOST")
	if pg_db_host != "" {
		result.PgDbHost = pg_db_host
	} else {
		logger.Info("PG_DB_HOST not set, using default value 127.0.0.1")
	}

	prometheus_server := os.Getenv("PROMETHEUS_SERVER")
	if prometheus_server != "" {
		result.PrometheusServer = prometheus_server
	} else {
		logger.Info("PROMETHEUS_SERVER not set, using default value http://127.0.0.1:8080")
	}

	return result
}
