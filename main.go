package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/statestorage"
	"github.com/pivotal-cf/brokerapi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// releaseVersion should be set by the linker at compile time.
var releaseVersion = "development-build"

// Default values for the configuration variables.
const (
	DefaultLogLevel = "INFO"

	DefaultAtlasBaseURL = "https://cloud.mongodb.com/api/atlas/v1.0/"
	DefaultRealmBaseURL = "https://realm.mongodb.com/api/admin/v3.0/"

	DefaultServerHost = "127.0.0.1"
	DefaultServerPort = 4000
)

func main() {
	// Add --help and -h flag.
	helpDescription := "Print information about the MongoDB Atlas Service Broker and helpful links."
	helpFlag := flag.Bool("help", false, helpDescription)
	flag.BoolVar(helpFlag, "h", false, helpDescription)

	// Add --version and -v flag.
	versionDescription := "Print current version of MongoDB Atlas Service Broker."
	versionFlag := flag.Bool("version", false, versionDescription)
	flag.BoolVar(versionFlag, "v", false, versionDescription)

	flag.Parse()

	// Output help message if help flag was specified.
	if *helpFlag {
		fmt.Println(getHelpMessage())
		return
	}

	// Output current version if version flag was specified.
	if *versionFlag {
		fmt.Println(releaseVersion)
		return
	}

	startBrokerServer()
}

func getHelpMessage() string {
	const helpMessage = `MongoDB Atlas Service Broker %s

This is a Service Broker which provides access to MongoDB deployments running
in MongoDB Atlas. It conforms to the Open Service Broker specification and can
be used with any compatible platform, for example the Kubernetes Service Catalog.

For instructions on how to install and use the Service Broker please refer to
the documentation: https://docs.mongodb.com/atlas-open-service-broker

Github: https://github.com/mongodb/mongodb-atlas-service-broker
Docker Image: quay.io/mongodb/mongodb-atlas-service-broker`

	return fmt.Sprintf(helpMessage, releaseVersion)
}

func deduceCredentials(logger *zap.SugaredLogger, atlasURL string) *credentials.Credentials {
	logger.Info("Deducing credentials source...")

	logger.Info("Trying Multi-Project credentials from env...")
	creds, err := credentials.FromEnv(atlasURL)
	switch {
	case err == nil && creds == nil:
		logger.Infow("Rejected Multi-Project (env): not enabled by user")
	case err == nil:
		logger.Info("Selected Multi-Project (env)")
		return creds
	default:
		logger.Fatalw("Error while loading env credentials", "error", err)
	}

	logger.Info("Trying Multi-Project credentials from CredHub...")
	creds, err = credentials.FromCredHub(atlasURL)
	switch {
	case err == nil && creds == nil:
		logger.Infow("Rejected Multi-Project (CredHub): not in CF")
	case err == nil:
		logger.Info("Selected Multi-Project (CredHub)")
		return creds
	default:
		logger.Fatalw("Error while loading CredHub credentials", "error", err)
	}

	logger.Info("Selected Basic Auth")
	logger.Fatal("Basic Auth credentials are not implemented yet")
	return nil
}

func createCredsAndDB(logger *zap.SugaredLogger, atlasURL string, realmURL string) (creds *credentials.Credentials, state *statestorage.RealmStateStorage) {
	creds = deduceCredentials(logger, atlasURL)

	if err := creds.FlattenOrgs(atlasURL); err != nil {
		logger.Fatalw("Cannot parse Org API Keys", "error", err)
	}

	id, _ := creds.RandomKey()
	ss, err := statestorage.GetStateStorage(creds, atlasURL, realmURL, logger, id)
	if err != nil {
		logger.Fatalw("Failed to get statestorage", "error", err)
	}

	logger.Debugw("GetOrgStateStorage", "ss.RealmApp", ss.RealmApp)

	return creds, ss
}

func createBroker(logger *zap.SugaredLogger) *broker.Broker {
	atlasURL := getEnvOrDefault("ATLAS_BASE_URL", DefaultAtlasBaseURL)
	realmURL := getEnvOrDefault("REALM_BASE_URL", DefaultRealmBaseURL)

	creds, state := createCredsAndDB(logger, atlasURL, realmURL)

	// Administrators can control what providers/plans are available to users
	pathToWhitelistFile, hasWhitelist := os.LookupEnv("PROVIDERS_WHITELIST_FILE")
	if !hasWhitelist {
		logger.Infow("Creating broker", "atlas_base_url", atlasURL, "whitelist_file", "NONE")
		return broker.New(logger, creds, atlasURL, nil, state)
	}

	// TODO
	logger.Fatal("Whitelist is not implemented yet")

	whitelist, err := broker.ReadWhitelistFile(pathToWhitelistFile)
	if err != nil {
		logger.Fatal("Cannot load providers whitelist: %v", err)
	}

	logger.Infow("Creating broker", "atlas_base_url", atlasURL, "whitelist_file", pathToWhitelistFile)
	return broker.New(logger, creds, atlasURL, whitelist, state)
}

func startBrokerServer() {
	logLevel := getEnvOrDefault("BROKER_LOG_LEVEL", DefaultLogLevel)
	logger, err := createLogger(logLevel)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := logger.Sync() // Flushes buffer, if any
		if err != nil {
			panic(err)
		}
	}()

	b := createBroker(logger)

	router := mux.NewRouter()
	brokerapi.AttachRoutes(router, b, NewLagerZapLogger(logger))

	// The auth middleware will convert basic auth credentials into an Atlas
	// client.
	router.Use(b.AuthMiddleware())

	// Configure TLS from environment variables.
	tlsEnabled, tlsCertPath, tlsKeyPath := getTLSConfig(logger)

	host := getEnvOrDefault("BROKER_HOST", DefaultServerHost)
	port := getIntEnvOrDefault("BROKER_PORT", getIntEnvOrDefault("PORT", DefaultServerPort))

	// Replace with NONE if not set
	logger.Infow("Starting API server", "releaseVersion", releaseVersion, "host", host, "port", port, "tls_enabled", tlsEnabled)

	// Start broker HTTP server.
	address := host + ":" + strconv.Itoa(port)

	var serverErr error
	if tlsEnabled {
		serverErr = http.ListenAndServeTLS(address, tlsCertPath, tlsKeyPath, router)
	} else {
		logger.Warn("TLS is disabled")
		serverErr = http.ListenAndServe(address, router)
	}

	if serverErr != nil {
		logger.Fatal(serverErr)
	}
}

func getTLSConfig(logger *zap.SugaredLogger) (bool, string, string) {
	certPath := getEnvOrDefault("BROKER_TLS_CERT_FILE", "")
	keyPath := getEnvOrDefault("BROKER_TLS_KEY_FILE", "")

	hasCertPath := certPath != ""
	hasKeyPath := keyPath != ""

	// Bail if only one of the cert and key has been provided.
	if (hasCertPath && !hasKeyPath) || (!hasCertPath && hasKeyPath) {
		logger.Fatal("Both a certificate and private key are necessary to enable TLS")
	}

	return hasCertPath && hasKeyPath, certPath, keyPath
}

// getEnvOrDefault will try getting an environment variable and return a default
// value in case it doesn't exist.
func getEnvOrDefault(name string, def string) string {
	value, exists := os.LookupEnv(name)
	if !exists {
		return def
	}

	return value
}

// getIntEnvOrDefault will try getting an environment variable and parse it as
// an integer. In case the variable is not set it will return the default value.
func getIntEnvOrDefault(name string, def int) int {
	value, exists := os.LookupEnv(name)
	if !exists {
		return def
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf(`Environment variable "%s" is not an integer`, name))
	}

	return intValue
}

// createLogger will create a zap sugared logger with the specified log level.
func createLogger(levelName string) (*zap.SugaredLogger, error) {
	levelByName := map[string]zapcore.Level{
		"DEBUG": zapcore.DebugLevel,
		"INFO":  zapcore.InfoLevel,
		"WARN":  zapcore.WarnLevel,
		"ERROR": zapcore.ErrorLevel,
	}

	// Convert log level string to a zap level.
	level, ok := levelByName[levelName]
	if !ok {
		return nil, fmt.Errorf(`invalid log level "%s"`, levelName)
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	// https://github.com/uber-go/zap/issues/584
	config.OutputPaths = []string{"stdout"}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}
