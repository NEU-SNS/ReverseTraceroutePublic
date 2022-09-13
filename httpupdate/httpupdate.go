/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package httpupdate

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/blang/semver"
)

type Status struct {
	Version string `json:"version"`
	Get     string `json:"get"`
}

type updater struct {
	statUrl      string
	version      string
	fetchUrl     string
	newVersion   string
	shouldUpdate bool
}

var DefaultUpdater updater

func (u *updater) FetchUrl() string {
	return u.fetchUrl
}

func (u *updater) NewVersion() string {
	return u.newVersion
}

func (u *updater) Version() string {
	return u.version
}

func (u *updater) CheckUpdate(statUrl, version string) (bool, error) {
	u.version = version
	u.statUrl = statUrl
	resp, err := http.Get(statUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	stat := &Status{}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Failed to get status: %d", http.StatusOK)
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(stat)
	if err != nil {
		return false, err
	}
	u.fetchUrl = stat.Get
	vr, err := semver.Make(version)
	if err != nil {
		// Return true if the version passed in isn't a semvar
		// Works around the case when plvp --version fails for older versions
		return true, nil
	}
	vn, err := semver.Make(stat.Version)
	if err != nil {
		return false, err
	}
	if vr.Compare(vn) < 0 {
		u.shouldUpdate = true
		return true, nil
	}
	return false, nil
}

func CheckUpdate(statUrl, version string) (bool, error) {
	return DefaultUpdater.CheckUpdate(statUrl, version)
}

func FetchUrl() string {
	return DefaultUpdater.FetchUrl()
}

func NewVersion() string {
	return DefaultUpdater.NewVersion()
}

func Version() string {
	return DefaultUpdater.Version()
}
