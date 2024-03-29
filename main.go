package main

import (
	"context"
	"encoding/json"
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
	s.OpenAPI.Info.Version = "v1.0.0"

	// Setup middlewares.
	s.Wrap(
		gzip.Middleware,
	)

	s.Use(cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "https://s3rbug.github.io"},
		AllowedMethods: []string{
			http.MethodGet,
		},
		AllowedHeaders:   []string{"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"},
		AllowCredentials: false,
	}).Handler)
	s.HandleFunc("/favicon.ico", getFavicon)

	s.Get("/matches/{id}", getGlyphsByID())

	s.Get("/matches", getMatches())

	s.Get("/matches/", getMatches())

	s.Docs("/docs", v4emb.New)

	log.Println("Starting service")
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), s); err != nil {
		log.Fatal(err)
	}
}

func getMatches() usecase.Interactor {
	var Demos []string
	u := usecase.NewIOI(nil, Demos, func(ctx context.Context, _, output interface{}) error {
		filename := "match_ids.json"
		file, err := ioutil.ReadFile(filename)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		err = json.Unmarshal(file, &Demos)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		out := output.(*[]string)
		*out = Demos
		return nil
	})
	u.SetTags("Matches")
	return u
}

func getFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/favicon.ico")
}

func getGlyphsByID() usecase.Interactor {
	var glyphs []structs.Glyph

	type getGlyphByIDInput struct {
		ID string `path:"id"`
	}

	u := usecase.NewIOI(getGlyphByIDInput{}, glyphs, func(ctx context.Context, input, output interface{}) error {
		in := input.(getGlyphByIDInput)

		match_id := (in.ID)

		fmt.Println("Requested MatchId: " + match_id)

		stateOfMatchID, err := utils.IsDownloadedDemo(match_id)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		if stateOfMatchID == "Downloaded" {
			// Downloading demo file
			sb, err := utils.GetMatchStructWithMatchID(match_id)

			if err != nil {
				return status.Wrap(err, status.InvalidArgument)
			}

			filename_bz2 := fmt.Sprintf("%v.dem.bz2", match_id)
			filename := filename_bz2[:len(filename_bz2)-4]

			err = utils.RetrieveFileWithURL(sb, filename_bz2)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}

			fmt.Printf("Match %d is downloaded\n", sb[0].Match_id)

			glyphs, err = parser.ParseDemo(filename, match_id)

			if err != nil {
				return status.Wrap(err, status.Internal)
			}

			err = os.Remove("dem_files/" + filename)

			if err != nil {
				return status.Wrap(err, status.Internal)
			}

		} else if stateOfMatchID == "None" {
			filename := "parsed_matches/" + match_id + ".json"
			file, err := ioutil.ReadFile(filename)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}

			err = json.Unmarshal(file, &glyphs)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}

		} else if stateOfMatchID == "Downloading" {
			fmt.Printf("%v is being parsed\n", match_id)
			return status.Wrap(err, status.Unavailable)
		}

		fmt.Printf("File %v is parsed\n", match_id)
		utils.AppendDownloadedDemo(match_id)

		out := output.(*[]structs.Glyph)
		*out = glyphs

		return nil
	})
	u.SetTags("Glyphs")
	u.SetExpectedErrors(status.NotFound)

	return u
}
