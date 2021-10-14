package config

import "time"

const (
	CFEventuallyTimeoutDefault   = 60 * time.Second
	CFConsistentlyTimeoutDefault = 60 * time.Millisecond
	CFEventuallyTimeoutMiddle    = 10 * time.Minute
	IntervalMiddle               = 10 * time.Second

	// cf timouts
	CFStagingTimeout  = 15
	CFStartingTimeout = 15

	TKey            = "testKey" // TODO get it from the plan
	MarketPlaceName = "atlas"
	TestPath        = "./test/cfe2e/data"

	// cloudqa
	CloudQAHost  = "https://cloud-qa.mongodb.com/api/atlas/v1.0/"
	CloudQARealm = "https://realm-qa.mongodb.com/api/admin/v3.0/"

	// test application coordinates
	TestAppRepo = "https://github.com/leo-ri/simple-ruby.git"
)
