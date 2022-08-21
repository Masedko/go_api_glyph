package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/Masedko/go_api_glyph/parser"
	"github.com/Masedko/go_api_glyph/structs"
	"github.com/Masedko/go_api_glyph/utils"
	"github.com/rs/cors"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/swgui/v4emb"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

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

	s.Get("/matches", getMatches())

	s.Docs("/docs", v4emb.New)

	log.Println("Starting service")
	if err := http.ListenAndServe("localhost:8080", s); err != nil {
		log.Fatal(err)
	}
}

func getMatches() usecase.Interactor {
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
	u := usecase.NewIOI(nil, Demos, func(ctx context.Context, _, output interface{}) error {
		out := output.(*[]string)
		*out = Demos
		return nil
	})
	u.SetTags("Matches")
	return u
}

func getGlyphsByID() usecase.Interactor {
	var glyphs []structs.Glyph

	type getGlyphByIDInput struct {
		ID string `path:"id"`
	}

	u := usecase.NewIOI(getGlyphByIDInput{}, glyphs, func(ctx context.Context, input, output interface{}) error {
		in := input.(getGlyphByIDInput)

		match_id := (in.ID)

		match_correctness := utils.CheckMatchIDCorrectness(match_id)

		if !match_correctness {
			return status.Wrap(errors.New("Match_id wrong type"), status.NotFound)
		}

		fmt.Println("Requested MatchId: " + match_id)

		filename := match_id + ".dem"

		if utils.IsDownloadedDemo(match_id) {
			// Downloading demo file
			sb := utils.GetMatchStructWithMatchID(match_id)

			URL_demo := fmt.Sprintf("http://replay%d.valve.net/570/%d_%d.dem.bz2", sb[0].Cluster, sb[0].Match_id, sb[0].Replay_salt)

			utils.RetrieveFileWithURL(URL_demo, sb, filename)

			fmt.Printf("File %d.dem is downloaded\n", sb[0].Match_id)

			glyphs = parser.ParseDemo(filename, match_id)

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

		out := output.(*[]structs.Glyph)
		*out = glyphs

		return nil
	})
	u.SetTags("Glyphs")
	u.SetExpectedErrors(status.NotFound)

	return u
}
