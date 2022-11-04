package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/Masedko/go_api_glyph/structs"
	"github.com/machinebox/graphql"
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
	matchId, err := strconv.Atoi(match_id)
	if err != nil {
		return nil, err
	}
	client := graphql.NewClient("https://api.stratz.com/graphql")
	req := graphql.NewRequest(`
    query($key: Long!) {
		match(id: $key) {
		  replaySalt
		  clusterId
		}
	}`)
	req.Var("key", matchId)
	// set any variables
	stratzToken := "Bearer " + "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1laWQiOiJodHRwczovL3N0ZWFtY29tbXVuaXR5LmNvbS9vcGVuaWQvaWQvNzY1NjExOTgzMzQwMzE4MTQiLCJ1bmlxdWVfbmFtZSI6ItCS0L4g0YHQu9Cw0LLRgyDQv9C70LXRgtC4ISIsIlN1YmplY3QiOiJiOTFjNDAxNy1iYzQwLTQ4NTMtOGJiMC03YmZkNzgyNTU1MDYiLCJTdGVhbUlkIjoiMzczNzY2MDg2IiwibmJmIjoxNjQzNTAwMjg2LCJleHAiOjE2NzUwMzYyODYsImlhdCI6MTY0MzUwMDI4NiwiaXNzIjoiaHR0cHM6Ly9hcGkuc3RyYXR6LmNvbSJ9.FPtVZnsflLsNMhM7VtL9qJkB6B9SwOpaWAJFII-jHiM"
	req.Header.Set("Authorization", stratzToken)

	// define a Context for the request
	ctx := context.Background()

	// run it and capture the response
	graphqlRequest := make(map[string]map[string]int)
	if err := client.Run(ctx, req, &graphqlRequest); err != nil {
		return nil, err
	}
	sb := []structs.Match{}
	match := structs.Match{
		Match_id:    matchId,
		Cluster:     graphqlRequest["match"]["clusterId"],
		Replay_salt: graphqlRequest["match"]["replaySalt"]}
	sb = append(sb, match)
	fmt.Println(sb)
	return sb, nil
}

func RetrieveFileWithURL(sb []structs.Match, filename string) error {
	URL_demo := fmt.Sprintf("http://replay%d.valve.net/570/%d_%d.dem.bz2", sb[0].Cluster, sb[0].Match_id, sb[0].Replay_salt)
	fmt.Println(URL_demo)
	ctx := context.Background()

	dl := got.NewDownload(ctx, URL_demo, "dem_files/"+filename)

	// Init
	if err := dl.Init(); err != nil {
		return err
	}
	fmt.Println("Start download")
	// Start download
	if err := dl.Start(); err != nil {
		return err
	}
	fmt.Println("Decompressing bzip2 file")
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
