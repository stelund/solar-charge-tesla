package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type CurrentPower struct {
	Power float64 `json:"power"`
}

type Energy struct {
	Energy float64 `json:"energy"`
}

type SiteOverview struct {
	LastUpdateTime string       `json:"lastUpdateTime"`
	LifeTimeData   Energy       `json:"lifeTimeData"`
	LastYearData   Energy       `json:"lastYearData"`
	LastMonthData  Energy       `json:"lastMonthData"`
	LastDayData    Energy       `json:"lastDayData"`
	CurrentPower   CurrentPower `json:"currentPower"`
	MeasuredBy     string       `json:"measuredBy"`
}

type SiteEnergy struct {
	SiteId       int          `json:"siteId"`
	SiteOverview SiteOverview `json:"siteOverview"`
}

type SitesOverviews struct {
	Count          int          `json:"count"`
	SiteEnergyList []SiteEnergy `json:"siteEnergyList"`
}

type OverviewResponse struct {
	SitesOverviews SitesOverviews `json:"sitesOverviews"`
}

type solarEdgeClient struct {
	apiKey string
	siteId int
}

func (s solarEdgeClient) getCurrentPower() (float64, error) {
	var o OverviewResponse
	v := url.Values{}
	v.Set("api_key", s.apiKey)
	u := url.URL{
		Scheme:   "https",
		Host:     "monitoringapi.solaredge.com",
		Path:     fmt.Sprintf("sites/%d/overview", s.siteId),
		RawQuery: v.Encode(),
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&o)
	if err != nil {
		return 0, err
	}
	if o.SitesOverviews.Count == 0 {
		return 0, errors.New("No sites overviews found")
	}
	return o.SitesOverviews.SiteEnergyList[0].SiteOverview.CurrentPower.Power, nil
}
