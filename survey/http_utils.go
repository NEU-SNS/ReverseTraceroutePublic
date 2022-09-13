package survey

import(
	"fmt"
	"net/http"
	"errors"
)


func QueryMeasurement(url string, authKey string) (*http.Response, error) {
	// Now query the measurement result
	// log.Info(fmt.Sprintf("Trying to fetch results at %s", url))
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Api-Key", authKey)
	req.Header.Set("Revtr-Key", authKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		message := fmt.Sprintf("Bad http status code %d while trying to fetch results %s ", 
		resp.StatusCode, url)
		return nil, errors.New(message) 
	}
	return resp, err 
}