package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/umahmood/haversine"
	"google.golang.org/api/iterator"
)

type site struct {
	Name                string    `firestore:"name"`
	Vendor              string    `firestore:"vendor"`
	SiteId              int       `firestore:"siteId"`
	ApiKey              string    `firestore:"apikey"`
	LastUpdated         time.Time `firestore:"lastUpdated"`
	SolarPower          float64   `firestore:"solarPower"`
	StartChargeTreshold float64   `firestore:"startChargeTreshold"`
	StopChargeTreshold  float64   `firestore:"stopChargeTreshold"`
	Longitude           float64   `firestore:"longitude"`
	Latitude            float64   `firestore:"latitude"`
}

type car struct {
	Name         string    `firestore:"name"`
	Vendor       string    `firestore:"vendor"`
	CarID        int64     `firestore:"carId"`
	AccessToken  string    `firestore:"accessToken"`
	RefreshToken string    `firestore:"refreshToken"`
	LastUpdated  time.Time `firestore:"lastUpdated"`
	BatteryLevel int32     `firestore:"batteryLevel"`
	Longitude    float64   `firestore:"longitude"`
	Latitude     float64   `firestore:"latitude"`
	IsCharging   bool      `firestore:"isCharging"`
}

func SolarChargeTesla(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	app := createApp(ctx)
	defer app.close()
	sites, err := readSites(app, ctx)
	if err != nil {
		log.Fatalf("Failed to read sites: %v", err)
	}

	poweredSites := []site{}
	for _, s := range sites {
		if s.SolarPower > 3000 {
			poweredSites = append(poweredSites, s)
		}
	}
	if len(poweredSites) == 0 {
		fmt.Fprintf(w, "state: idle")
	}

	cars, err := readCars(app, ctx)
	if err != nil {
		log.Fatalf("Failed to read cars: %v", err)
	}

	charging := evolve(app, sites, cars)
	fmt.Fprintf(w, "charging: %s", html.EscapeString(strconv.Itoa(charging)))
}

func main() {
	ctx := context.Background()

	app := createApp(ctx)
	defer app.close()

	sites, err := readSites(app, ctx)
	if err != nil {
		log.Fatalf("Failed to read sites: %v", err)
	}
	fmt.Printf("sites %v\n", sites)
	cars, err := readCars(app, ctx)
	if err != nil {
		log.Fatalf("Failed to read cars: %v", err)
	}
	fmt.Printf("cars %v\n", cars)
}

func nextState(a solarChargeTesla, s site, c car) (error) {
	client, err := a.createCarClient(c)
	if err != nil {
		return err
	}
	if !c.IsCharging && s.SolarPower > s.StartChargeTreshold {
		return client.startCharging(c.CarID)
	} else if c.IsCharging && s.SolarPower < s.StopChargeTreshold {
		return client.stopCharging(c.CarID)
	}
	return nil
}

func evolve(a solarChargeTesla, sites []site, cars []car) int {
	charging := 0
	for _, s := range sites {
		siteCoord := haversine.Coord{Lat: s.Latitude, Lon: s.Longitude}
		for _, c := range cars {
			carCoord := haversine.Coord{Lat: c.Latitude, Lon: c.Longitude}
			_, km := haversine.Distance(siteCoord, carCoord)
			if km < 0.01 {
				err := nextState(a, s, c)
				if err != nil {
					fmt.Printf("Error for car %d: %v+", c.CarID, err)
				}
			}
		}
	}
	return charging
}

type solarClient interface {
	getCurrentPower() (float64, error)
}

type carClient interface {
	getCarData(CarID int64) (*carData, error)
	startCharging(CarID int64) error
	stopCharging(CarID int64) error
}

type solarChargeTesla interface {
	createSolarClient(site) (solarClient, error)
	createCarClient(car) (carClient, error)
	close() error
	getFirestoreClient() *firestore.Client
}

type realApp struct {
	fc *firestore.Client
}

func createApp(ctx context.Context) *realApp {
	projectID := os.Getenv("GCP_PROJECT_ID")

	if projectID == "" {
		projectID = "default"
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	app := realApp{fc: client}
	return &app
}

func (a realApp) createSolarClient(s site) (solarClient, error) {
	if s.Vendor == "SolarEdge" {
		return solarEdgeClient{siteId: s.SiteId, apiKey: s.ApiKey}, nil
	}
	return nil, errors.New(fmt.Sprintf("Unknown site vendor %s", s.Vendor))
}

func (a realApp) createCarClient(c car) (carClient, error) {
	if c.Vendor == "Tesla" {
		return teslaClient{apiClient: teslaAPIClient{accessToken: c.AccessToken}}, nil
	}
	return nil, errors.New(fmt.Sprintf("Unknown car vendor %s", c.Vendor))
}

func (a realApp) getFirestoreClient() *firestore.Client {
	return a.fc
}

func (a realApp) close() error {
	return a.fc.Close()
}

func readSites(app solarChargeTesla, ctx context.Context) ([]site, error) {
	iter := app.getFirestoreClient().Collection("sites").Documents(ctx)
	sites := []site{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var s site
		doc.DataTo(&s)
		if time.Now().UTC().After(s.LastUpdated.Add(time.Hour * 1)) {
			sar, err := app.createSolarClient(s)
			if err != nil {
				log.Fatalf("Failed to create site client: %v", err)
			}
			power, err := sar.getCurrentPower()
			if err == nil {
				s.SolarPower = power
				s.LastUpdated = time.Now().UTC()
			}
		}
		sites = append(sites, s)
	}
	return sites, nil
}

func readCars(app solarChargeTesla, ctx context.Context) ([]car, error) {
	iter := app.getFirestoreClient().Collection("cars").Documents(ctx)
	cars := []car{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var c car
		doc.DataTo(&c)
		if c.LastUpdated.IsZero() || time.Now().UTC().After(c.LastUpdated.Add(time.Hour*1)) {
			fmt.Printf("Updating %v+", c.LastUpdated)
			cc, err := app.createCarClient(c)
			if err != nil {
				fmt.Printf("Failed to create tesla client: %v\n", err)
				continue
			}
			carData, err := cc.getCarData(c.CarID)
			if err == nil {
				c.BatteryLevel = carData.BatteryLevel
				c.Longitude = carData.Longitude
				c.Latitude = carData.Latitude
				c.LastUpdated = time.Now().UTC()
			} else {
				fmt.Printf("Failed to read tesla battery level: %v\n", err)
			}
		}
		cars = append(cars, c)
	}
	return cars, nil
}
