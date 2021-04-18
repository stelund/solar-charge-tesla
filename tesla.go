package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

type vehiclesData struct {
	VehicleID int64  `json:"vehicle_id"`
	State     string `json:"state"`
	InService bool   `json:"in_service"`
}

type vehiclesDataResponse struct {
	Vehicles []vehiclesData `json:"response"`
}

type chargeStates struct {
	BatteryHeaterOn             bool    `json:"battery_heater_on"`
	BatteryLevel                int32   `json:"battery_level"`
	BatteryRange                float64 `json:"battery_range"`
	ChargeCurrentRequest        int32   `json:"charge_current_request"`
	ChargeCurrentRequestMax     int32   `json:"charge_current_request_max"`
	ChargeEnableRequest         bool    `json:"charge_enable_request"`
	ChargeEnergyAdded           float64 `json:"charge_energy_added"`
	ChargeLimitSoc              int32   `json:"charge_limit_soc"`
	ChargeLimitSocMax           int32   `json:"charge_limit_soc_max"`
	ChargeLimitSocMin           int32   `json:"charge_limit_soc_min"`
	ChargeLimitSocStd           int32   `json:"charge_limit_soc_std"`
	ChargeMilesAddedIdeal       float64 `json:"charge_miles_added_ideal"`
	ChargeMilesAddedRated       float64 `json:"charge_miles_added_rated"`
	ChargePortColdWeatherMode   string  `json:"charge_port_cold_weather_mode"`
	ChargePortDoorOpen          bool    `json:"charge_port_door_open"`
	ChargePortLatch             string  `json:"charge_port_latch"`
	ChargeRate                  float64 `json:"charge_rate"`
	ChargeToMaxRange            bool    `json:"charge_to_max_range"`
	ChargerActualCurrent        int32   `json:"charger_actual_current"`
	ChargerPhases               int32   `json:"charger_phases"`
	ChargerPilotCurrent         int32   `json:"charger_pilot_current"`
	ChargerPower                int32   `json:"charger_power"`
	ChargerVoltage              int32   `json:"charger_voltage"`
	ChargingState               string  `json:"charging_state"`
	ConnChargeCable             string  `json:"conn_charge_cable"`
	EstBatteryRange             float64 `json:"est_battery_range"`
	FastChargerBrand            string  `json:"fast_charger_brand"`
	FastChargerPresent          bool    `json:"fast_charger_present"`
	FastChargerType             string  `json:"fast_charger_type"`
	IdealBatteryRange           float64 `json:"ideal_battery_range"`
	ManagedChargingActive       bool    `json:"managed_charging_active"`
	ManagedChargingStartTime    string  `json:"managed_charging_start_time"`
	ManagedChargingUserCanceled bool    `json:"managed_charging_user_canceled"`
	MaxRangeChargeCounter       int32   `json:"max_range_charge_counter"`
	NotEnoughPowerToHeat        bool    `json:"not_enough_power_to_heat"`
	ScheduledChargingPending    bool    `json:"scheduled_charging_pending"`
	ScheduledChargingStartTime  string  `json:"scheduled_charging_start_time"`
	TimeToFullCharge            float64 `json:"time_to_full_charge"`
	Timestamp                   int64   `json:"timestamp"`
	TripCharging                bool    `json:"trip_charging"`
	UsableBatteryLevel          int32   `json:"usable_battery_level"`
	UserChargeEnableRequest     string  `json:"user_charge_enable_request"`
}

type driveStates struct {
	GpsAsOf                 int32   `json:"gps_as_of"`
	Heading                 int32   `json:"heading"`
	Latitude                float64 `json:"latitude"`
	Longitude               float64 `json:"longitude"`
	NativeLatitude          float64 `json:"native_latitude"`
	NativeLocationSupported int32   `json:"native_location_supported"`
	NativeLongitude         float64 `json:"native_longitude"`
	NativeType              string  `json:"native_type"`
	Power                   int32   `json:"power"`
	ShiftState              string  `json:"shift_state"`
	Speed                   string  `json:"speed"`
	Timestamp               int64   `json:"timestamp"`
}

type teslaCarData struct {
	DisplayName string       `json:"display_name"`
	State       string       `json:"state"`
	ChargeState chargeStates `json:"charge_state"`
	DriveState  driveStates  `json:"drive_state"`
}

type vehicleDataResponse struct {
	Car teslaCarData `json:"response"`
}

type teslaAPIClient struct {
	accessToken string
}

type carAPIClient interface {
	makeRequest(string, string) (*http.Response, error)
}

func (t teslaAPIClient) makeRequest(method string, path string) (*http.Response, error) {
	client := &http.Client{}
	u := url.URL{
		Scheme: "https",
		Host:   "owner-api.teslamotors.com",
		Path:   path,
	}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return &http.Response{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.accessToken))
	return client.Do(req)
}

type wakeData struct {
	State string `json:"state"`
}

type wakeDataResponse struct {
	Wake wakeData `json:"response"`
}

func wakeCar(cac carAPIClient, carID int64) error {
	var w wakeDataResponse
	for i := 1; i < 15; i++ {
		resp, err := cac.makeRequest("POST", fmt.Sprintf("/api/1/vehicles/%d/wake", carID))
		if err != nil {
			return errors.Wrap(err, "posting to wake endpoint")
		}
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&w)
		if err != nil {
			return err
		}
		if w.Wake.State == "online" {
			return nil
		}
		time.Sleep(2)
	}
	return errors.New(fmt.Sprintf("Car is not waking. Still in state %s", w.Wake.State))
}

func getCarState(cac carAPIClient, carID int64) (string, error) {
	resp, err := cac.makeRequest("GET", "/api/1/vehicles")
	if err != nil {
		return "", errors.Wrap(err, "fetching vehicles")
	}
	if resp.StatusCode >= 400 {
		return "", errors.New(fmt.Sprintf("Status code %d when fetching vehicles", resp.StatusCode))
	}
	defer resp.Body.Close()
	var vr vehiclesDataResponse
	err = json.NewDecoder(resp.Body).Decode(&vr)
	if err != nil {
		return "", errors.Wrap(err, "parsing vehicles")
	}
	for _, v := range vr.Vehicles {
		if v.VehicleID == carID {
			return v.State, nil
		}
	}
	return "", errors.New(fmt.Sprintf("Unable to find vehicle with id %d", carID))
}

func ensureAwake(cac carAPIClient, carID int64) error {
	if state, err := getCarState(cac, carID); err != nil {
		return err
	} else if state == "online" {
		return nil
	}
	return wakeCar(cac, carID)
}

type carData struct {
	BatteryLevel int32
	Longitude    float64
	Latitude     float64
}

type teslaClient struct {
	apiClient carAPIClient
}

func (t teslaClient) getCarData(carID int64) (*carData, error) {
	if err := ensureAwake(t.apiClient, carID); err != nil {
		return nil, errors.Wrap(err, "waking car")
	}
	resp, err := t.apiClient.makeRequest("GET", fmt.Sprintf("/api/1/vehicles/%d/vehicle_data", carID))
	if err != nil {
		return nil, errors.Wrap(err, "fetching vehicle_data")
	}
	defer resp.Body.Close()
	var v vehicleDataResponse
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return nil, errors.Wrap(err, "parsing vehicle_data response")
	}
	return &carData{
		BatteryLevel: v.Car.ChargeState.BatteryLevel,
		Longitude:    v.Car.DriveState.Longitude,
		Latitude:     v.Car.DriveState.Latitude,
	}, nil
}

func (t teslaClient) startCharging(carID int64) error {
	return errors.New("Not implemented")
}

func (t teslaClient) stopCharging(carID int64) error {
	return errors.New("Not implemented")
}
