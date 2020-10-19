// Copyright 2020 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/alexflint/go-arg"
	"github.com/gorilla/mux"
	"github.com/mongodb/atlas-osb/pkg/broker"
	"github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"github.com/pivotal-cf/brokerapi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const toolName = "atlas-aosb"

// releaseVersion should be set by the linker at compile time.
var releaseVersion = "0.0.0+devbuild." + getBinaryFootprint()

// command-line arguments and env variables with default values
type Args struct {
	LogLevel  zapcore.Level `arg:"-l,env:BROKER_TLS_KEY_FILE" default:"INFO"`
	SentryDSN string        `arg:"env:SENTRY_DSN"`

	BrokerConfig
}

type BrokerConfig struct {
	AtlasURL            string `arg:"-a,env:ATLAS_BASE_URL" default:"https://cloud.mongodb.com/api/atlas/v1.0/"`
	RealmURL            string `arg:"-r,env:REALM_BASE_URL" default:"https://realm.mongodb.com/api/admin/v3.0/"`
	Host                string `arg:"-h,env:BROKER_HOST" default:"127.0.0.1"`
	Port                uint16 `arg:"-p,env:BROKER_PORT" default:"4000"`
	CertPath            string `arg:"-c,env:BROKER_TLS_CERT_FILE"`
	KeyPath             string `arg:"-k,env:BROKER_TLS_KEY_FILE"`
	ServiceName         string `arg:"env:BROKER_OSB_SERVICE_NAME" default:"atlas"`
	ServiceDisplayName  string `arg:"env:BROKER_OSB_SERVICE_DISPLAY_NAME" default:"Template Services"`
	ServiceDesc         string `arg:"env:BROKER_OSB_SERVICE_DESC" default:"MongoDB Atlas Plan Template Deployments"`
	ServiceTags         string `arg:"env:BROKER_OSB_SERVICE_TAGS" default:"mongodb"`
	ImageURL            string `arg:"env:BROKER_OSB_IMAGE_URL" default:"https://webassets.mongodb.com/_com_assets/cms/vectors-anchor-circle-mydmar539a.svg"`
	DocumentationURL    string `arg:"env:BROKER_OSB_DOCS_URL" default:"https://support.mongodb.com/welcome"`
	ProviderDisplayName string `arg:"env:BROKER_OSB_PROVIDER_DISPLAY_NAME" default:"MongoDB"`
	LongDescription     string `arg:"env:BROKER_OSB_PROVIDER_DESC" default:"Complete MongoDB Atlas deployments managed through resource templates. See https://github.com/mongodb/atlas-osb"`
}

// FIXME: update links
func (*Args) Description() string {
	const helpMessage = `This is a Service Broker which provides access to MongoDB deployments running
in MongoDB Atlas. It conforms to the Open Service Broker specification and can
be used with any compatible platform, for example the Kubernetes Service Catalog.

For instructions on how to install and use the Service Broker please refer to
the documentation: https://TBD

Github: https://TBD
Docker Image: https://TBD
`

	return helpMessage
}

func (*Args) Version() string {
	return fmt.Sprintf("MongoDB Atlas Service Broker v%s", releaseVersion)
}

var args Args

func main() {
	p := arg.MustParse(&args)

	hasCertPath := args.CertPath != ""
	hasKeyPath := args.KeyPath != ""
	// Bail if only one of the cert and key has been provided.
	if hasCertPath != hasKeyPath {
		p.Fail("Both a certificate and private key are necessary to enable TLS")
	}

	startBrokerServer()
}

func deduceCredentials(logger *zap.SugaredLogger, atlasURL string) *credentials.Credentials {
	logger.Info("Deducing credentials source...")

	logger.Info("Trying Multi-Project credentials from env...")
	creds, err := credentials.FromEnv(atlasURL)
	switch {
	case err == nil && creds == nil:
		logger.Infow("Rejected Multi-Project (env): no credentials in env")
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

func createBroker(logger *zap.SugaredLogger) *broker.Broker {
	logger.Infow("Creating broker", "atlas_base_url", args.AtlasURL)

	creds := deduceCredentials(logger, args.AtlasURL)
	userAgent := fmt.Sprintf("%s/%s (%s;%s)", toolName, releaseVersion, runtime.GOOS, runtime.GOARCH)

	return broker.New(logger, creds, broker.Config(args.BrokerConfig), userAgent)
}

func startBrokerServer() {
	logger, err := createLogger()
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

	tlsEnabled := args.CertPath != ""

	// Replace with NONE if not set
	logger.Infow("Starting API server", "releaseVersion", releaseVersion, "host", args.Host, "port", args.Port, "tls", tlsEnabled)

	// Start broker HTTP server.
	address := args.Host + ":" + fmt.Sprint(args.Port)

	var serverErr error
	if tlsEnabled {
		serverErr = http.ListenAndServeTLS(address, args.CertPath, args.KeyPath, router)
	} else {
		logger.Warn("TLS is disabled")
		serverErr = http.ListenAndServe(address, router)
	}

	if serverErr != nil {
		logger.Fatal(serverErr)
	}
}

func addSentryLogger(log *zap.Logger, dsn string) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level: zapcore.WarnLevel, //when to send message to sentry
		Tags: map[string]string{
			"component":      "system",
			"releaseVersion": releaseVersion,
		},
	}

	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromDSN(dsn))
	//in case of err it will return noop core. so we can safely attach it
	if err != nil {
		log.Warn("failed to init zap", zap.Error(err))
	}
	return zapsentry.AttachCoreToLogger(core, log)
}

// createLogger will create a zap sugared logger with the specified log level.
func createLogger() (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.Level.SetLevel(args.LogLevel)
	// https://github.com/uber-go/zap/issues/584
	config.OutputPaths = []string{"stdout"}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	if args.SentryDSN != "" {
		logger = addSentryLogger(logger, args.SentryDSN)
	}

	return logger.Sugar(), nil
}

func getBinaryFootprint() string {
	fname := os.Args[0]
	f, err := ioutil.ReadFile(fname)
	if err != nil {
		return "unknown"
	}

	cs := sha256.Sum256(f)
	bcs := hex.EncodeToString(cs[:])
	return bcs[:16]
}
