package util

type UniqueRand struct {
    Generated map[int]bool
}

type ProbesResponse struct {
	// "count": 34003,
	// "next": "https://atlas.ripe.net/api/v2/probes/?page=2",
	// "previous": null,
	// "results": [
	//   {
	// 	"address_v4": "45.138.229.91",
	// 	"address_v6": "2a10:3781:e22:1:220:4aff:fec8:23d7",
	// 	"asn_v4": 206238,
	// 	"asn_v6": 206238,
	// 	"country_code": "NL",
	// 	"description": "Robert #1 100/10 XS4All",
	// 	"first_connected": 1288367583,
	// 	"geometry": {
	// 	  "type": "Point",
	// 	  "coordinates": [
	// 		4.9275,
	// 		52.3475
	// 	  ]
	// 	},
	// 	"id": 1,
	// 	"is_anchor": false,
	// 	"is_public": true,
	// 	"last_connected": 1618859012,
	// 	"prefix_v4": "45.138.228.0/22",
	// 	"prefix_v6": "2a10:3780::/29",
	// 	"status": {
	// 	  "id": 1,
	// 	  "name": "Connected",
	// 	  "since": "2021-04-09T08:40:48Z"
	// 	},
	Count int `json:"count,omitempty"`
	Next *string `json:"next,omitempty"`
	ProbesResults []ProbeResponse `json:"results,omitempty"`

}

type ProbeResponse struct {
	ID int `json:"id,omitempty"`
	AddressV4 string `json:"address_v4,omitempty"`
	IsAnchor bool `json:"is_anchor,omitempty"`
	Status ProbeStatus `json:"status,omitempty"`
}

type ProbeStatus struct {
	ID int `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type AnchorsResponse struct {
	// "https://atlas.ripe.net/api/v2/anchors"
	//"count": 888,
	//"next": "https://atlas.ripe.net/api/v2/anchors/?page=2",
	//"previous": null,
	Count int `json:"count,omitempty"`
	Next *string `json:"next,omitempty"`
	AnchorsResults []AnchorResponse `json:"results,omitempty"`
}

type AnchorResponse struct {
	ID int `json:"id,omitempty"`
	Fqdn string `json:"fqdn,omitempty"`
	ProbeID int `json:"probe,omitempty"`
	IPv4 string `json:"ip_v4,omitempty"`
	ASv4 int `json:"as_v4,omitempty"`
	City string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	IsDisabled bool `json:"is_disabled,omitempty"`


	/**

	id (integer): The id of the anchor,
type (string): The type of the object,
fqdn (string): The fully qualified domain name of the anchor,
probe (integer): The id of the probe that is hosted on this anchor,
is_ipv4_only (boolean),
ip_v4 (string): The IPv4 address (if any) of this anchor,
as_v4 (integer): The IPv4 AS this anchor belongs to,
ip_v4_gateway (string): The IPv4 gateway address of this anchor,
ip_v4_netmask (string): The IPv4 netmask for the IP address of this anchor,
ip_v6 (string): The IPv6 address (if any) of this anchor,
as_v6 (integer): The IPv6 AS this anchor belongs to,
ip_v6_gateway (string): The IPv6 gateway address of this anchor,
ip_v6_prefix (integer): The IPv6 prefix of this anchor,
city (string): The city this anchor is located in,
country (string): An ISO-3166-1 alpha-2 code indicating the country that this probe is located in, as derived from the user supplied longitude and latitude,
geometry (number): A GeoJSON point object containing the location of this anchor. The longitude and latitude are contained within the `coordinates` array,
tlsa_record (string): Installed TLSA DNS resource record on this anchor,
is_disabled (boolean),
date_live (string),
hardware_version (integer) = ['0' or '1' or '2' or '99']
	**/
}