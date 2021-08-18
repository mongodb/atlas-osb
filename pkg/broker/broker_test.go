package broker

import (
	"encoding/base64"
	"testing"

	"github.com/pkg/errors"
)

func TestDecodePlan(t *testing.T) {
	t.Run("Plan with current version of the apiKey field", func(t *testing.T) {
		planStr := `{"apiKey": {"desc": "API Key for Atlas OSB", 
				"id": "atlas-osb-api-key",
				"privateKey": "<key>",
				"publicKey": "<ksy>",
				"orgId": "<orgid>"},
			"cluster": {"labels": [{"key": "Infrastructure Tool",
						"value": "MongoDB Atlas Service Broker"}],
				"name": "gitlabEmpRpt-svc-new",
				"providerBackupEnabled": true,
				"providerSettings": {"diskTypeName": "P4",
					"instanceSizeName": "M10",
					"providerName": "AZURE",
					"regionName": "CANADA_CENTRAL"}}, 
			"description": "A dedicated, single region, M10 tier MongoDB Atlas Cluster (1 vCPU, 2GB RAM, 32GB Storage, 120 IOPS)",
			"free": true,
			"ipWhitelists": [{"comment": "everything", "ipAddress": "0.0.0.0/1"},
				{"comment": "everything", "ipAddress": "128.0.0.0/1"}],
			"name": "M10-small",
			"project": {"id": "<id>",
				"name": "gitlabEmpRpt-svc-new",
				"orgId": "<id>"},
			"settings": {"overrideBindDB": "default", "overrideBindDBRole": "readWrite"}}`

		planEnc := base64.StdEncoding.EncodeToString([]byte(planStr))

		plan, err := decodePlan(planEnc)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Log(plan)

		if plan.APIKey == nil || plan.Name == "" {
			t.Fatal(errors.New("Failed to parse the plan"))
		}

		if plan.APIKey["orgID"] == "" {
			t.Fatal(errors.New("Failed to parse apiKey.orgID from the plan"))
		}
	})

	t.Run("Plan with an old version of the apiKey field", func(t *testing.T) {
		planStrOld := `{"apiKey": {"desc": "API Key for Atlas OSB", 
				"id": "atlas-osb-api-key",
				"privateKey": "<key>",
				"publicKey": "<ksy>",
				"roles": [{"orgId": "<orgid>"}]},
			"cluster": {"labels": [{"key": "Infrastructure Tool",
						"value": "MongoDB Atlas Service Broker"}],
				"name": "gitlabEmpRpt-svc-new",
				"providerBackupEnabled": true,
				"providerSettings": {"diskTypeName": "P4",
					"instanceSizeName": "M10",
					"providerName": "AZURE",
					"regionName": "CANADA_CENTRAL"}}, 
			"description": "A dedicated, single region, M10 tier MongoDB Atlas Cluster (1 vCPU, 2GB RAM, 32GB Storage, 120 IOPS)",
			"free": true,
			"ipWhitelists": [{"comment": "everything", "ipAddress": "0.0.0.0/1"},
				{"comment": "everything", "ipAddress": "128.0.0.0/1"}],
			"name": "M10-small",
			"project": {"id": "<id>",
				"name": "gitlabEmpRpt-svc-new",
				"orgId": "<id>"},
			"settings": {"overrideBindDB": "default", "overrideBindDBRole": "readWrite"}}`

		planEnc := base64.StdEncoding.EncodeToString([]byte(planStrOld))

		plan, err := decodePlan(planEnc)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Log(plan)

		if plan.APIKey["orgID"] == "" {
			t.Fatal(errors.New("Failed to parse apiKey.orgID from the plan"))
		}
	})
}
