package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

type fakeResponse struct {
	Path       string
	StatusCode int
	Body       string
}

type fakeTeslaClient struct {
	Responses []fakeResponse
}

func (fta fakeTeslaClient) makeRequest(method string, path string) (*http.Response, error) {
	for _, r := range fta.Responses {
		if r.Path == path {
			body := ioutil.NopCloser(bytes.NewReader([]byte(r.Body)))
			return &http.Response{
				StatusCode: r.StatusCode,
				Body:       body,
			}, nil
		}
	}
	return &http.Response{
		StatusCode: 404,
	}, nil
}

func TestWakeCar(t *testing.T) {
	tests := []struct {
		state   string
		isError bool
	}{
		{state: "online", isError: false},
		{state: "offline", isError: true},
	}

	for _, test := range tests {
		fta := &fakeTeslaClient{
			[]fakeResponse{
				{
					Path:       "/api/1/vehicles/1234/wake",
					StatusCode: 200,
					Body: fmt.Sprintf(`{
						  		   "response": {
									  "state": "%s"
									}
								  }`, test.state),
				},
			},
		}
		err := wakeCar(fta, 1234)
		if test.isError && err == nil {
			t.Errorf("Expected err but was %v+", err)
		}
		if !test.isError && err != nil {
			t.Errorf("Expected err but was %v+", err)
		}
	}
}

func TestGetCarState(t *testing.T) {
	tests := []struct {
		state string
	}{
		{state: "online"},
		{state: "offline"},
	}

	for _, test := range tests {
		fta := &fakeTeslaClient{
			[]fakeResponse{
				{
					Path:       "/api/1/vehicles",
					StatusCode: 200,
					Body: fmt.Sprintf(`{
									"response": [{
										"vehicle_id": 1235,
										"state": "unknown",
										"in_service": false
										},
										{
										"vehicle_id": 1234,
										"state": "%s",
										"in_service": false
										}]
									}`, test.state),
				},
			},
		}
		state, err := getCarState(fta, 1234)
		if err != nil {
			t.Errorf("Didnt expect error fetching car state %v+", err)
		}
		if test.state != state {
			t.Errorf("Expected state %s but was %s", test.state, state)
		}
	}
}

func TestGetCarData(t *testing.T) {
	tests := []struct {
		batteryLevel int32
		longitude    float64
		latitude     float64
	}{
		{batteryLevel: 20, longitude: 1.1121, latitude: 1.2123},
		{batteryLevel: 40, longitude: 12.1121, latitude: 10.2123},
	}

	for _, test := range tests {
		fta := &fakeTeslaClient{
			[]fakeResponse{
				{
					Path:       "/api/1/vehicles",
					StatusCode: 200,
					Body: `{
							"response": [
								{
									"vehicle_id": 1234,
									"state": "online",
									"in_service": false
								}
							]
						}`,
				},
				{
					Path:       "/api/1/vehicles/1234/vehicle_data",
					StatusCode: 200,
					Body: fmt.Sprintf(`{
						"response": {
						  "id": 12345678901234567,
						  "user_id": 123,
						  "vehicle_id": 1234,
						  "vin": "5YJSA11111111111",
						  "display_name": "Nikola 2.0",
						  "option_codes": "AD15,MDL3,PBSB,RENA,BT37,ID3W,RF3G,S3PB,DRLH,DV2W,W39B,APF0,COUS,BC3B,CH07,PC30,FC3P,FG31,GLFR,HL31,HM31,IL31,LTPB,MR31,FM3B,RS3H,SA3P,STCP,SC04,SU3C,T3CA,TW00,TM00,UT3P,WR00,AU3P,APH3,AF00,ZCST,MI00,CDM0",
						  "color": null,
						  "access_type": "OWNER",
						  "tokens": ["abcdef1234567890", "1234567890abcdef"],
						  "state": "online",
						  "in_service": false,
						  "id_s": "12345678901234567",
						  "calendar_enabled": true,
						  "api_version": 13,
						  "backseat_token": null,
						  "backseat_token_updated_at": null,
						  "drive_state": {
							"gps_as_of": 1607623884,
							"heading": 5,
							"latitude": %f,
							"longitude": %f,
							"native_latitude": 33.111111,
							"native_location_supported": 1,
							"native_longitude": -88.111111,
							"native_type": "wgs",
							"power": -9,
							"shift_state": null,
							"speed": null,
							"timestamp": 1607623897515
						  },
						  "climate_state": {
							"battery_heater": false,
							"battery_heater_no_power": false,
							"climate_keeper_mode": "off",
							"defrost_mode": 0,
							"driver_temp_setting": 21.1,
							"fan_status": 0,
							"inside_temp": 22.1,
							"is_auto_conditioning_on": false,
							"is_climate_on": false,
							"is_front_defroster_on": false,
							"is_preconditioning": false,
							"is_rear_defroster_on": false,
							"left_temp_direction": -66,
							"max_avail_temp": 28.0,
							"min_avail_temp": 15.0,
							"outside_temp": 18.0,
							"passenger_temp_setting": 21.1,
							"remote_heater_control_enabled": false,
							"right_temp_direction": -66,
							"seat_heater_left": 0,
							"seat_heater_right": 0,
							"side_mirror_heaters": false,
							"timestamp": 1607623897515,
							"wiper_blade_heater": false
						  },
						  "charge_state": {
							"battery_heater_on": false,
							"battery_level": %d,
							"battery_range": 149.92,
							"charge_current_request": 40,
							"charge_current_request_max": 40,
							"charge_enable_request": true,
							"charge_energy_added": 2.42,
							"charge_limit_soc": 90,
							"charge_limit_soc_max": 100,
							"charge_limit_soc_min": 50,
							"charge_limit_soc_std": 90,
							"charge_miles_added_ideal": 10.0,
							"charge_miles_added_rated": 8.0,
							"charge_port_cold_weather_mode": null,
							"charge_port_door_open": true,
							"charge_port_latch": "Engaged",
							"charge_rate": 28.0,
							"charge_to_max_range": false,
							"charger_actual_current": 40,
							"charger_phases": 1,
							"charger_pilot_current": 40,
							"charger_power": 9,
							"charger_voltage": 243,
							"charging_state": "Charging",
							"conn_charge_cable": "SAE",
							"est_battery_range": 132.98,
							"fast_charger_brand": "<invalid>",
							"fast_charger_present": false,
							"fast_charger_type": "<invalid>",
							"ideal_battery_range": 187.4,
							"managed_charging_active": false,
							"managed_charging_start_time": null,
							"managed_charging_user_canceled": false,
							"max_range_charge_counter": 0,
							"minutes_to_full_charge": 165,
							"not_enough_power_to_heat": false,
							"scheduled_charging_pending": false,
							"scheduled_charging_start_time": null,
							"time_to_full_charge": 2.75,
							"timestamp": 1607623897515,
							"trip_charging": false,
							"usable_battery_level": 59,
							"user_charge_enable_request": null
						  },
						  "gui_settings": {
							"gui_24_hour_time": false,
							"gui_charge_rate_units": "mi/hr",
							"gui_distance_units": "mi/hr",
							"gui_range_display": "Rated",
							"gui_temperature_units": "F",
							"show_range_units": true,
							"timestamp": 1607623897515
						  },
						  "vehicle_state": {
							"api_version": 13,
							"autopark_state_v2": "standby",
							"autopark_style": "standard",
							"calendar_supported": true,
							"car_version": "2020.48.10 f8900cddd03a",
							"center_display_state": 0,
							"df": 0,
							"dr": 0,
							"fd_window": 0,
							"fp_window": 0,
							"ft": 0,
							"homelink_device_count": 2,
							"homelink_nearby": true,
							"is_user_present": false,
							"last_autopark_error": "no_error",
							"locked": false,
							"media_state": { "remote_control_enabled": true },
							"notifications_supported": true,
							"odometer": 57869.762487,
							"parsed_calendar_supported": true,
							"pf": 0,
							"pr": 0,
							"rd_window": 0,
							"remote_start": false,
							"remote_start_enabled": true,
							"remote_start_supported": true,
							"rp_window": 0,
							"rt": 0,
							"sentry_mode": false,
							"sentry_mode_available": true,
							"smart_summon_available": true,
							"software_update": {
							  "download_perc": 0,
							  "expected_duration_sec": 2700,
							  "install_perc": 1,
							  "status": "",
							  "version": ""
							},
							"speed_limit_mode": {
							  "active": false,
							  "current_limit_mph": 85.0,
							  "max_limit_mph": 90,
							  "min_limit_mph": 50,
							  "pin_code_set": false
							},
							"summon_standby_mode_enabled": false,
							"sun_roof_percent_open": 0,
							"sun_roof_state": "closed",
							"timestamp": 1607623897515,
							"valet_mode": false,
							"valet_pin_needed": true,
							"vehicle_name": null
						  },
						  "vehicle_config": {
							"can_accept_navigation_requests": true,
							"can_actuate_trunks": true,
							"car_special_type": "base",
							"car_type": "models2",
							"charge_port_type": "US",
							"default_charge_to_max": false,
							"ece_restrictions": false,
							"eu_vehicle": false,
							"exterior_color": "White",
							"has_air_suspension": true,
							"has_ludicrous_mode": false,
							"motorized_charge_port": true,
							"plg": true,
							"rear_seat_heaters": 0,
							"rear_seat_type": 0,
							"rhd": false,
							"roof_color": "None",
							"seat_type": 2,
							"spoiler_type": "None",
							"sun_roof_installed": 2,
							"third_row_seats": "None",
							"timestamp": 1607623897515,
							"trim_badging": "p90d",
							"use_range_badging": false,
							"wheel_type": "AeroTurbine19"
						  }
						}
					  }`, test.latitude, test.longitude, test.batteryLevel),
				},
			},
		}
		cc := teslaClient{apiClient: fta}
		carData, err := cc.getCarData(1234)
		if err != nil {
			t.Errorf("Didnt expect error fetching car data %v+", err)
		}
		if test.batteryLevel != carData.BatteryLevel {
			t.Errorf("Expected state %d but was %d", test.batteryLevel, carData.BatteryLevel)
		}
		if test.longitude != carData.Longitude {
			t.Errorf("Expected state %f but was %f", test.longitude, carData.Longitude)
		}
		if test.latitude != carData.Latitude {
			t.Errorf("Expected state %f but was %f", test.latitude, carData.Latitude)
		}
	}
}
