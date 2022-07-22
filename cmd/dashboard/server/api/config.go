package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch/v5"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
)

// SettingsHandler handles post settings call
func (a *Api) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		content, err := a.getCurrentSettings()
		if err != nil {
			responseServerError(w, err)
			return
		} else {
			w.Write(content)
		}
	case "POST":
		defer r.Body.Close()
		content, err := ioutil.ReadAll(r.Body)
		if err != nil {
			responseServerError(w, err)
			return
		}
		_, err = a.updateCurrentSettings(content)
		if err != nil {
			responseServerError(w, err)
			return
		}
	}
}

// getCurrentSettings returns current config json string
func (a *Api) getCurrentSettings() ([]byte, error) {
	return json.Marshal(a.config)
}

// updateCurrentSettings patch current config
func (a *Api) updateCurrentSettings(patch []byte) ([]byte, error) {
	// TODO mask readonly fields
	current, err := a.getCurrentSettings()
	if err != nil {
		return nil, err
	}
	after, err := jsonpatch.MergePatch(current, patch)
	if err != nil {
		return nil, err
	}
	newConf := common.Config{}
	err = json.Unmarshal(after, &newConf)
	if err != nil {
		return nil, err
	}
	err = a.config.Update(newConf)
	if err != nil {
		return nil, err
	}
	return after, nil
}
