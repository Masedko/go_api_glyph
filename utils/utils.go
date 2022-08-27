package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/Masedko/go_api_glyph/structs"
	"github.com/melbahja/got"
)

func parseMatch(jsonBuffer []byte) ([]structs.Match, error) {

	match := []structs.Match{}

	err := json.Unmarshal(jsonBuffer, &match)
	if err != nil {
		return nil, err
	}

	return match, nil
}

func GetMatchStructWithMatchID(match_id string) ([]structs.Match, error) {
	URL_id := "https://api.opendota.com/api/replays?match_id=" + match_id
	resp, err := http.Get(URL_id)

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	sb, err := parseMatch(body)
	if err != nil {
		return nil, err
	}
	if len(sb) == 0 {
		return nil, errors.New("OpenDota returned empty match :(")
	}
	return sb, nil
}

func RetrieveFileWithURL(sb []structs.Match, filename string) error {
	URL_demo := fmt.Sprintf("http://replay%d.valve.net/570/%d_%d.dem.bz2", sb[0].Cluster, sb[0].Match_id, sb[0].Replay_salt)
	ctx := context.Background()

	dl := got.NewDownload(ctx, URL_demo, "dem_files/"+filename)

	// Init
	if err := dl.Init(); err != nil {
		return err
	}

	// Start download
	if err := dl.Start(); err != nil {
		return err
	}

	app := "bzip2"
	arg0 := "-d"
	arg1 := "dem_files/" + filename

	cmd := exec.Command(app, arg0, arg1)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func StringInSlice(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func IsDownloadedDemo(match_id string) (string, error) {
	state := "None"
	var Demos []string
	filename := "match_ids.json"
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(file, &Demos)
	if err != nil {
		return state, err
	}
	if !StringInSlice(Demos, match_id) {
		state = "Downloaded"
	}
	if _, err := os.Stat("dem_files/" + match_id + ".dem"); err == nil {
		state = "Downloading"
	}
	return state, nil
}

func AppendDownloadedDemo(match_id string) error {
	var Demos []string
	filename := "match_ids.json"
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &Demos)
	if err != nil {
		return err
	}
	if !StringInSlice(Demos, match_id) {
		Demos = append(Demos, match_id)
		file, err = json.Marshal(Demos)
		if err != nil {
			return err
		}
		_ = ioutil.WriteFile("match_ids.json", file, 0644)
	}
	return nil
}
