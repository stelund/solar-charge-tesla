package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

type testApp struct {
	fc                *firestore.Client
	s                 *testSolarVendor
	initialSolarPower float64
}

func createTestApp(ctx context.Context, initialSolarPower float64) *testApp {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		log.Fatal("No firestore emulator running, start with:\ngcloud beta emulators firestore start")
	}

	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		log.Fatalf("firebase.NewClient err: %v", err)
	}

	return &testApp{
		fc:                client,
		initialSolarPower: initialSolarPower,
	}
}

type testSolarVendor struct {
	solarPower float64
}

func (s testSolarVendor) getCurrentPower() (float64, error) {
	return s.solarPower, nil
}

func (a testApp) createSolarClient(s site) (solarClient, error) {
	if s.Vendor == "TestSolarVendor" {
		a.s = &testSolarVendor{a.initialSolarPower}
		return a.s, nil
	}
	return nil, errors.New(fmt.Sprintf("Unknown site vendor %s", s.Vendor))
}

type testCarVendor struct{}

func (c testCarVendor) getCarData(CarID int64) (*carData, error) {
	return nil, nil
}

func (c testCarVendor) stopCharging(CarID int64) error {
	return nil
}

func (c testCarVendor) startCharging(CarID int64) error {
	return nil
}

func (a testApp) createCarClient(c car) (carClient, error) {
	if c.Vendor == "TestCarVendor" {
		return testCarVendor{}, nil
	}
	return nil, errors.New(fmt.Sprintf("Unknown car vendor %s", c.Vendor))
}

func (a testApp) getFirestoreClient() *firestore.Client {
	return a.fc
}

func (a testApp) close() error {
	return a.fc.Close()
}

/*func TestSolarChargeTesla(t *testing.T) {
        tests := []struct {
                body string
                want string
        }{
                {body: `{"name": ""}`, want: "Hello, World!"},
                {body: `{"name": "Gopher"}`, want: "Hello, Gopher!"},
        }

        for _, test := range tests {
                req := httptest.NewRequest("GET", "/", strings.NewReader(test.body))
                req.Header.Add("Content-Type", "application/json")

                rr := httptest.NewRecorder()
                HelloHTTP(rr, req)

                if got := rr.Body.String(); got != test.want {
                        t.Errorf("HelloHTTP(%q) = %q, want %q", test.body, got, test.want)
                }
        }
}*/

/*


 */

func setupTestSite(a *testApp, ctx context.Context, solarPower float64, lastUpdated time.Time) {
	d := a.fc.Doc("sites/site1")
	d.Set(ctx, site{
		Name:        "Test",
		Vendor:      "TestSolarVendor",
		SiteId:      123,
		ApiKey:      "abcdefgh",
		Longitude:   100.0,
		Latitude:    120.0,
		SolarPower:  solarPower,
		LastUpdated: lastUpdated,
	})
}

func TestReadSites(t *testing.T) {
	ctx := context.Background()
	currentPower := 1000.0
	recordedPower := 800.0
	app := createTestApp(ctx, currentPower)
	defer app.close()
	tests := []struct {
		want        float64
		lastUpdated time.Time
	}{
		{want: recordedPower, lastUpdated: time.Now().UTC()},
		{want: currentPower, lastUpdated: time.Time{}},
		{want: currentPower, lastUpdated: time.Now().UTC().Add(time.Duration(-5 * time.Hour))},
	}

	for _, test := range tests {
		setupTestSite(app, ctx, recordedPower, test.lastUpdated)
		sites, err := readSites(app, ctx)
		if err != nil {
			t.Fatalf("Failed with err %v", err)
		}
		if len(sites) != 1 {
			t.Fatalf("Unwanted length of sites %v", sites)
		}
		if sites[0].SolarPower != test.want {
			t.Fatalf("Want %f got %f", test.want, sites[0].SolarPower)
		}
	}
}
