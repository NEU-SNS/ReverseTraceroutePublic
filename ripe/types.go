package ripe


type ProbesResponse struct {
	// "https://atlas.ripe.net/api/v2/anchors"
	//"count": 888,
	//"next": "https://atlas.ripe.net/api/v2/anchors/?page=2",
	//"previous": null,
	Count int `json:"count,omitempty"`
	Next *string `json:"next,omitempty"`
	ProbesResults []ProbeResponse `json:"results,omitempty"`
}

type ProbeResponse struct {
	ID int `json:"id,omitempty"`
	// Fqdn string `json:"fqdn,omitempty"`
	// ProbeID int `json:"id,omitempty"`
	IPv4 string `json:"address_v4,omitempty"`
	ASv4 int `json:"asn_v4,omitempty"`
	// City string `json:"city,omitempty"`
	// Country string `json:"country,omitempty"`
	// IsDisabled bool `json:"is_disabled,omitempty"`
}


type RIPEMeasurementsIDs struct {
	MeasurementIDs  [] int `json:"measurements"`      
}

type RIPEPollingMeasurement struct {
	Status RIPEMeasurementStatus  `json:"status"`
}

type RIPEMeasurementStatus struct {
	Name string  `json:"name"`
}

type RIPEMeasurementError struct {
	Error RIPEMeasurementErrorDetail `json:"error"`
}

type RIPEMeasurementErrorDetail struct {
	ErrorsDetails [] RIPEMeasurementErrorDetailError `json:"errors"`
}

type RIPEMeasurementErrorDetailError struct {
	Detail string `json:"detail"`
}