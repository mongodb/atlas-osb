package broker

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/pkg/errors"
)

const testDataDir = "../../test/data"

func TestDecodePlan(t *testing.T) {
	t.Run("Plan with current version of the apiKey field", func(t *testing.T) {
		planData, err := os.ReadFile(testDataDir + "/realmPlan.json")
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		planEnc := base64.StdEncoding.EncodeToString(planData)
		t.Log(planEnc)

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
		planData, err := os.ReadFile(testDataDir + "/realmPlanOld.json")
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		planEnc := base64.StdEncoding.EncodeToString(planData)

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
