package main

import (
	"compress/bzip2"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	valid "github.com/asaskevich/govalidator"
	"github.com/dotabuff/manta"
	"github.com/dotabuff/manta/dota"
	"github.com/rs/cors"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/swgui/v4emb"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"golang.org/x/exp/slices"
)

type Glyph struct {
	User_name    string `json:"user_name" description:"Username (not current)"`
	User_steamID uint64 `json:"user_steamID" description:"Steam64 ID"`
	Minute       uint32 `json:"minute" minimum:"0" maximum:"60" description:"Minute when glyph was used"`
	Second       uint32 `json:"second" minimum:"0" maximum:"60" description:"Second when glyph was used"`
	HeroID       uint64 `json:"heroId" description:"ID of hero (https://liquipedia.net/dota2/MediaWiki:Dota2webapi-heroes.json)"`
}

type HeroPlayer struct {
	Hero_ID   int64
	Player_ID int64
}

type Match struct {
	Match_id    int `json:"match_id"`
	Cluster     int `json:"cluster"`
	Replay_salt int `json:"replay_salt"`
}

func main() {
	s := web.DefaultService()

	s.OpenAPI.Info.Title = "Glyph by MatchID API"
	s.OpenAPI.Info.WithDescription("This service provides API to get glyph usage in Dota 2 match based on match_id")
	s.OpenAPI.Info.Version = "v0.2.1"

	// Setup middlewares.
	s.Wrap(
		gzip.Middleware,
	)
	s.Use(cors.AllowAll().Handler)

	s.Get("/matches/{id}", getGlyphsByID())

	s.Docs("/docs", v4emb.New)

	log.Println("Starting service")
	if err := http.ListenAndServe("localhost:8080", s); err != nil {
		log.Fatal(err)
	}
}

func parseMatch(jsonBuffer []byte) ([]Match, error) {

	match := []Match{}

	err := json.Unmarshal(jsonBuffer, &match)
	if err != nil {
		return nil, err
	}

	return match, nil
}

func GetMatchStructWithMatchID(match_id string) []Match {
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

func RetrieveFileWithURL(URL_demo string, sb []Match, filename string) {
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

func ParseDemo(filename string, match_id string) []Glyph {
	f, err := os.Open("dem_files/" + filename)
	if err != nil {
		log.Fatalf("unable to open file: %s", err)
	}
	defer f.Close()

	p, err := manta.NewStreamParser(f)
	if err != nil {
		log.Fatalf("unable to create parser: %s", err)
	}

	gameStartTime := 0.0
	gameCurrentTime := 0.0
	var glyphs []Glyph
	var glyph Glyph
	var heroplayers []HeroPlayer
	for i := 0; i < 10; i++ {
		heroplayers = append(heroplayers, HeroPlayer{})
	}

	p.Callbacks.OnCDOTAUserMsg_SpectatorPlayerUnitOrders(func(m *dota.CDOTAUserMsg_SpectatorPlayerUnitOrders) error {
		if m.GetOrderType() == int32(dota.DotaunitorderT_DOTA_UNIT_ORDER_GLYPH) {
			mapEntity := p.FindEntity(m.GetEntindex()).Map()
			glyph = Glyph{
				User_name:    mapEntity["m_iszPlayerName"].(string),
				User_steamID: mapEntity["m_steamID"].(uint64),
				Minute:       uint32(gameCurrentTime-gameStartTime) / 60,
				Second:       uint32(math.Round(gameCurrentTime-gameStartTime)) % 60,
			}
			if !slices.Contains(glyphs, glyph) {
				glyphs = append(glyphs, glyph)
			}
		}
		return nil
	})

	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		if e.GetClassName() == "CDOTAGamerulesProxy" {
			gameStartTime, err = strconv.ParseFloat(fmt.Sprint(e.Map()["m_pGameRules.m_flGameStartTime"]), 64)
			gameCurrentTime, err = strconv.ParseFloat(fmt.Sprint(e.Map()["m_pGameRules.m_fGameTime"]), 64)
		}
		if gameCurrentTime < 700 && e.GetClassName() == "CDOTA_PlayerResource" {
			for i := 0; i < 10; i++ {
				heroplayers[i].Hero_ID, _ = strconv.ParseInt(fmt.Sprint(e.Map()["m_vecPlayerTeamData.000"+strconv.Itoa(i)+".m_nSelectedHeroID"]), 10, 64)
				heroplayers[i].Player_ID, _ = strconv.ParseInt(fmt.Sprint(e.Map()["m_vecPlayerData.000"+strconv.Itoa(i)+".m_iPlayerSteamID"]), 10, 64)
			}
		}
		return nil
	})

	p.Start()

	for k := range glyphs {
		for l := range heroplayers {
			if glyphs[k].User_steamID == uint64(heroplayers[l].Player_ID) {
				glyphs[k].HeroID = uint64(heroplayers[l].Hero_ID)
			}
		}
	}

	file, _ := json.MarshalIndent(glyphs, "", " ")
	if err != nil {
		log.Fatalln(err)
	}

	write_to := "parsed_matches/" + match_id + ".json"
	_ = ioutil.WriteFile(write_to, file, 0644)

	return glyphs
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

func getGlyphsByID() usecase.Interactor {
	var glyphs []Glyph

	type getGlyphByIDInput struct {
		ID string `path:"id"`
	}

	u := usecase.NewIOI(getGlyphByIDInput{}, glyphs, func(ctx context.Context, input, output interface{}) error {
		in := input.(getGlyphByIDInput)

		match_id := (in.ID)

		match_correctness := CheckMatchIDCorrectness(match_id)

		if !match_correctness {
			return status.Wrap(errors.New("Match_id wrong type"), status.NotFound)
		}

		fmt.Println("Requested MatchId: " + match_id)

		filename := match_id + ".dem"

		if IsDownloadedDemo(match_id) {
			// Downloading demo file
			sb := GetMatchStructWithMatchID(match_id)

			URL_demo := fmt.Sprintf("http://replay%d.valve.net/570/%d_%d.dem.bz2", sb[0].Cluster, sb[0].Match_id, sb[0].Replay_salt)

			RetrieveFileWithURL(URL_demo, sb, filename)

			fmt.Printf("File %d.dem is downloaded\n", sb[0].Match_id)

			glyphs = ParseDemo(filename, match_id)

			err := os.Remove("dem_files/" + filename)

			if err != nil {
				log.Fatalln(err)
			}

		} else {
			filename := "parsed_matches/" + match_id + ".json"
			file, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatalln(err)
			}
			err = json.Unmarshal(file, &glyphs)
			if err != nil {
				log.Fatalln(err)
			}
		}

		fmt.Printf("File %v is parsed\n", filename)

		out := output.(*[]Glyph)
		*out = glyphs

		return nil
	})
	u.SetTags("Glyphs")
	u.SetExpectedErrors(status.NotFound)

	return u
}
