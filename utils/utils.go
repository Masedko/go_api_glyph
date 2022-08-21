package utils

import (
	"compress/bzip2"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/Masedko/go_api_glyph/structs"
	valid "github.com/asaskevich/govalidator"
)

func parseMatch(jsonBuffer []byte) ([]structs.Match, error) {

	match := []structs.Match{}

	err := json.Unmarshal(jsonBuffer, &match)
	if err != nil {
		return nil, err
	}

	return match, nil
}

func GetMatchStructWithMatchID(match_id string) []structs.Match {
	URL_id := "https://api.opendota.com/api/replays?match_id=" + match_id
	resp, err := http.Get(URL_id)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	sb, err := parseMatch(body)
	if err != nil {
		log.Fatalln(err)
	}
	return sb
}

func RetrieveFileWithURL(URL_demo string, sb []structs.Match, filename string) {
	resp, err := http.Get(URL_demo)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalln(err)
	}
	r_bz2 := bzip2.NewReader(resp.Body)
	outfile, err := os.Create("dem_files/" + filename)
	defer outfile.Close()
	_, err = io.Copy(outfile, r_bz2)
}

func CheckMatchIDCorrectness(match_id string) bool {
	if valid.IsInt(match_id) {
		return true
	}
	return false
}

func StringInSlice(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func IsDownloadedDemo(match_id string) bool {
	IsDownloaded := false
	var Demos []string
	filename := "match_ids.json"
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(file, &Demos)
	if err != nil {
		log.Fatalln(err)
	}
	if !StringInSlice(Demos, match_id) {
		IsDownloaded = true
		Demos = append(Demos, match_id)
		file, err = json.Marshal(Demos)
		if err != nil {
			log.Fatalln(err)
		}
		_ = ioutil.WriteFile("match_ids.json", file, 0644)
	}
	return IsDownloaded
}
