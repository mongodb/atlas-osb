package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/pkg/errors"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		panic(errors.New("you need to pass a JSON file path as a second argument"))
	}

	path := args[1]
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("failed to read file %s: %w", path, err))
	}

	planEnc := string(data)
	if []rune(planEnc)[0] == '{' {
		planEnc = base64.StdEncoding.EncodeToString(data)
	}

	plan, err := decodePlan(planEnc)
	if err != nil {
		panic(fmt.Errorf("err: %w", err))
	}

	// fmt.Println(plan)

	if plan.APIKey == nil || plan.Name == "" {
		panic(errors.New("Failed to parse the plan"))
	}

	if plan.APIKey["orgID"] == "" {
		panic(errors.New("Failed to parse apiKey.orgID from the plan"))
	}

	fmt.Println("Testing passed ✅")
}

func decodePlan(enc string) (dynamicplans.Plan, error) {
	b64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(enc))
	dp := dynamicplans.Plan{}
	err := json.NewDecoder(b64).Decode(&dp)

	return dp, errors.Wrap(err, "cannot unmarshal plan")
}
