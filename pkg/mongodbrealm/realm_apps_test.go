package mongodbrealm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
    "os"
    "github.com/Sectorbob/mlab-ns2/gae/ns/digest"

	"github.com/go-test/deep"
)

/*

PUBLIC="DHVYWAQT"
PRIVATE="1a69ca3d-ace4-429f-b1e9-75e5475d75de"
group="5f12d8cc6c2bfd1e0c670f4a"
*/
const (
	publicKeyEnv  = "ATLAS_PUBLIC_KEY"
	privateKeyEnv = "ATLAS_PRIVATE_KEY"
	groupIDEnv      = "ATLAS_GROUP_ID"
)

var (
	publicKey  = os.Getenv(publicKeyEnv)
	privateKey = os.Getenv(privateKeyEnv)
	groupID      = os.Getenv(groupIDEnv)
)

const (
	// baseURLPath is a non-empty Client.BaseURL path to use during tests,
	// to ensure relative URLs are used for all endpoints.
	realmBaseURLPath = "/api-v1"
)
func realm_setup(c *Client,test *testing.T) {
    t := digest.NewTransport(publicKey, privateKey)
    tc, err := t.Client()
    if err != nil {
        test.Fatalf(err.Error())
    }

    c = NewClient(tc)

    c.SetCurrentRealmAtlasApiKey ( &RealmAtlasApiKey{
        Username: publicKey,
        Password: privateKey,
    })
    fmt.Printf("realm_setup ----> GetCurrentRealmAtlasApiKey() %v",c.GetCurrentRealmAtlasApiKey())
}

func TestRealmApps_ListRealmApps(t *testing.T) {
	client, mux, teardown := setup()
    realm_setup(client,t)
	defer teardown()

	mux.HandleFunc(fmt.Sprintf("/groups/%s/apps",groupID), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `[
        {
		  "_id": "111aaa111aaa111aaa111aaa",
          "client_app_id": "client-app-id-1",
          "name": "test-realm-app-1",
          "location": "US-VA",
          "deployment_model": "GLOBAL",
          "domain_id": "1",
          "group_id": "1",
          "last_used": 1595072140,
          "last_modified": 1595072140,
          "product": "standard"
        },
        {
		  "_id": "222bbb222bbb222bbb222bbb",
          "client_app_id": "client-app-id-2",
          "name": "test-realm-app-2",
          "location": "US-CA",
          "deployment_model": "GLOBAL",
          "domain_id": "1",
          "group_id": "1",
          "last_used": 1595072140,
          "last_modified": 1595072140,
          "product": "standard"
        }
        ]`)
	})

	apps, _, err := client.RealmApps.List(ctx, groupID, nil)

	if err != nil {
		t.Fatalf("RealmApps.List returned error: %v", err)
	}

	expected := []RealmApp{
		{
			ID:         "111aaa111aaa111aaa111aaa",
            ClientAppID: "client-app-id-1",
            Name: "test-realm-app-1",
            Location: "US-VA",
            DeploymentModel: "GLOBAL",
            DomainID: "1",
            GroupID: "1",
		},
		{
			ID:         "222bbb222bbb222bbb222bbb",
            ClientAppID: "client-app-id-2",
            Name: "test-realm-app-2",
            Location: "US-CA",
            DeploymentModel: "GLOBAL",
            DomainID: "1",
            GroupID: "2",
		},
        /*
            LastUsed: 1595072140,
            LastModified: 1595072140,
            Product: "standard",
            */
	}
	if diff := deep.Equal(apps, expected); diff != nil {
		t.Error(diff)
	}
}

func TestRealmApps_Create(t *testing.T) {
	client, mux, teardown := setup()
    realm_setup(client,t)
	defer teardown()


	createRequest := &RealmAppInput{
		Name:  "test-realm-app-3",
        ClientAppID: "client-app-id-3",
        Location: "US-VT",

	}

	mux.HandleFunc(fmt.Sprintf("/groups/%s/apps", groupID), func(w http.ResponseWriter, r *http.Request) {
		expected := map[string]interface{}{
            "name":  "test-realm-app-3",
            "client_app_id": "client-app-id-3",
            "location": "US-VT",
		}

		jsonBlob := `
        {
          "_id": "333ccc333ccc333ccc333ccc",
          "client_app_id": "client-app-id-3",
          "name": "test-realm-app-3",
          "location": "US-VT",
          "deployment_model": "GLOBAL",
          "domain_id": "1",
          "group_id": "1",
          "last_used": 1595072140,
          "last_modified": 1595072140,
          "product": "standard"
        }
        `

		var v map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		if diff := deep.Equal(v, expected); diff != nil {
			t.Error(diff)
		}

		fmt.Fprint(w, jsonBlob)
	})

	realmApp, _, err := client.RealmApps.Create(ctx, groupID, createRequest)
	if err != nil {
		t.Errorf("RealmApps.Create returned error: %v", err)
	}
    t.Log(fmt.Sprintf("realmApp: %+v",realmApp))
	if name := realmApp.Name; name != "test-realm-app-3" {
		t.Errorf("expected name '%s', received '%s'", "test-realm-app-3", name)
	}

	if location := realmApp.Location; location != "US-VT" {
		t.Errorf("expected location '%s', received '%s'", groupID, location)
	}
}

func TestRealmApps_GetRealmApp(t *testing.T) {
	client, mux, teardown := setup()
    realm_setup(client,t)
	defer teardown()

	mux.HandleFunc("/groups/1/apps/111aaa111aaa111aaa111aaa", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"name":"test-name"}`)
	})

	apps, _, err := client.RealmApps.Get(ctx, "1", "111aaa111aaa111aaa111aaa")
	if err != nil {
		t.Errorf("RealmApp.Get returned error: %v", err)
	}

	expected := &RealmApp{Name: "test-realm-app-1"}

	if diff := deep.Equal(apps, expected); diff != nil {
		t.Errorf("realmapps.Get = %v", diff)
	}
}

func TestRealmApps_Update(t *testing.T) {
	client, mux, teardown := setup()
    realm_setup(client,t)
	defer teardown()


	updateRequest := &RealmAppInput{
		Name:  "test-realm-app-1",
		Location:  "MARKS",
	}

	mux.HandleFunc(fmt.Sprintf("/groups/%s/apps/%s", groupID, "111aaa111aaa111aaa111aaa"), func(w http.ResponseWriter, r *http.Request) {
		expected := map[string]interface{}{
			"name":  "test-realm-app-1",
			"location":  "MARS",
		}

        jsonBlob :=`
        {
		  "_id": "111aaa111aaa111aaa111aaa",
          "client_app_id": "client-app-id-1",
          "name": "test-realm-app-1",
          "location": "MARS",
          "deployment_model": "GLOBAL",
          "domain_id": "1",
          "group_id": "1",
          "last_used": 1595072140,
          "last_modified": 1595072140,
          "product": "standard"
        }
        `

		var v map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		if diff := deep.Equal(v, expected); diff != nil {
			t.Error(diff)
		}

		fmt.Fprint(w, jsonBlob)
	})

	realmApp, _, err := client.RealmApps.Update(ctx, groupID, "111aaa111aaa111aaa111aaa", updateRequest)
	if err != nil {
		t.Fatalf("RealmApps.Create returned error: %v", err)
	}

	if name := realmApp.Name; name != "test-realm-app" {
		t.Errorf("expected name '%s', received '%s'", "test-realm-app-1", name)
	}

	if location := realmApp.Location; location != "MARS" {
		t.Errorf("expected location '%s', received '%s'", "MARS", location)
	}
}

func TestRealmApps_Delete(t *testing.T) {
	client, mux, teardown := setup()
    realm_setup(client,t)
	defer teardown()

	realmAppID := "111aaa111aaa111aaa111aaa"

	mux.HandleFunc(fmt.Sprintf("/groups/%s/apps/%s", groupID, realmAppID), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
	})


	_, err := client.RealmApps.Delete(ctx, groupID, realmAppID)
	if err != nil {
		t.Errorf("RealmApp.Delete returned error: %v", err)
	}
}
