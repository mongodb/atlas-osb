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
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/dynamicplans"
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

func deduceCredentials(logger *zap.SugaredLogger) *credentials.Credentials {
	logger.Info("Deducing credentials source...")

	logger.Info("Trying Multi-Project credentials from env...")
	creds, err := credentials.FromEnv()
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
	creds, err = credentials.FromCredHub()
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
	return nil
}

func deduceModeAndCreds(logger *zap.SugaredLogger, baseURL string) (broker.Mode, *credentials.Credentials) {
	logger.Info("Deducing catalog mode...")

	dynPlans := false
	autoPlans := false

	logger.Info("Trying Dynamic Plans...")
	p, err := dynamicplans.FromEnv()
	switch {
	case err == nil && p == nil:
		logger.Infow("Rejected Dynamic Plans", "reason", "ATLAS_BROKER_TEMPLATEDIR not set")
	case err == nil:
		logger.Info("Selected Dynamic Plans")
		dynPlans = true
	default:
		logger.Fatalw("Error while loading Dynamic Plans", "error", err)
	}

	logger.Info("Trying auto-generated plans...")
	autoPlans = getEnvOrDefault("BROKER_ENABLE_AUTOPLANSFROMPROJECTS", "") == "true"
	if autoPlans {
		logger.Info("Selected auto-generated plans")
	} else {
		logger.Info("Rejected auto-generated plans", "reason", "BROKER_ENABLE_AUTOPLANSFROMPROJECTS not set to 'true'")
		logger.Info("Selected static plans")
	}

	creds := deduceCredentials(logger)

	if creds == nil {
		if dynPlans {
			logger.Fatal("Cannot use dynamic plans without multi-project credentials")
		}
		if autoPlans {
			logger.Fatal("Cannot use auto-generated plans without multi-project credentials")
		}
	}

	if dynPlans {
		return broker.DynamicPlans, creds
	}

	if creds != nil {
		if err := creds.FlattenOrgs(baseURL); err != nil {
			logger.Fatalw("Cannot parse Org API Keys", "error", err)
		}

		if autoPlans {
			return broker.MultiGroupAutoPlans, creds
		}
		return broker.MultiGroup, creds
	}

	return broker.BasicAuth, nil
}

func createBroker(logger *zap.SugaredLogger) *broker.Broker {
	baseURL := getEnvOrDefault("ATLAS_BASE_URL", DefaultAtlasBaseURL)
	mode, creds := deduceModeAndCreds(logger, baseURL)

	// TODO: implement!
	if mode == broker.MultiGroup {
		logger.Fatal("Multi-group credentials used without BROKER_ENABLE_AUTOPLANSFROMPROJECTS: not implemented yet!")
	}

	// Administrators can control what providers/plans are available to users
	pathToWhitelistFile, hasWhitelist := os.LookupEnv("PROVIDERS_WHITELIST_FILE")
	if !hasWhitelist {
		logger.Infow("Creating broker", "atlas_base_url", baseURL, "whitelist_file", "NONE")
		return broker.New(logger, creds, baseURL, nil, mode)
	}

	whitelist, err := broker.ReadWhitelistFile(pathToWhitelistFile)
	if err != nil {
		logger.Fatal("Cannot load providers whitelist: %v", err)
	}

	logger.Infow("Creating broker", "atlas_base_url", baseURL, "whitelist_file", pathToWhitelistFile)
	return broker.New(logger, creds, baseURL, whitelist, mode)
}

func startBrokerServer() {
	logLevel := getEnvOrDefault("BROKER_LOG_LEVEL", DefaultLogLevel)
	logger, err := createLogger(logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // Flushes buffer, if any

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

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}
