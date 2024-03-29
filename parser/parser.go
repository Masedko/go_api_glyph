package parser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"

	"github.com/Masedko/go_api_glyph/structs"
	"github.com/dotabuff/manta"
	"github.com/dotabuff/manta/dota"
)

func ParseDemo(filename string, match_id string) ([]structs.Glyph, error) {
	f, err := os.Open("dem_files/" + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p, err := manta.NewStreamParser(f)
	if err != nil {
		return nil, err
	}
	gameStartTime := 0.0
	gameCurrentTime := 0.0
	var glyphs []structs.Glyph
	// var glyph structs.Glyph
	var heroplayers []structs.HeroPlayer
	for i := 0; i < 10; i++ {
		heroplayers = append(heroplayers, structs.HeroPlayer{})
	}
	p.Callbacks.OnCDOTAUserMsg_SpectatorPlayerUnitOrders(func(m *dota.CDOTAUserMsg_SpectatorPlayerUnitOrders) error {
		if m.GetOrderType() == int32(dota.DotaunitorderT_DOTA_UNIT_ORDER_GLYPH) {
			fmt.Println("Here")
			// mapEntity := p.FindEntity(m.GetEntindex()).Map()
			// glyph = structs.Glyph{
			// 	Username:     mapEntity["m_iszPlayerName"].(string),
			// 	User_steamID: fmt.Sprint(mapEntity["m_steamID"].(uint64)),
			// 	Minute:       uint32(gameCurrentTime-gameStartTime) / 60,
			// 	Second:       uint32(math.Round(gameCurrentTime-gameStartTime)) % 60,
			// }
			// if !slices.Contains(glyphs, glyph) {
			// 	glyphs = append(glyphs, glyph)
			// }
		}
		return nil
	})
	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		if e.GetClassName() == "CDOTAGamerulesProxy" {
			gameStartTime = float64(e.Get("m_pGameRules.m_flGameStartTime").(float32))
			gameCurrentTime = float64(e.Get("m_pGameRules.m_fGameTime").(float32))
			fmt.Println(gameCurrentTime, gameStartTime)
		}
		if gameCurrentTime < 1100 && e.GetClassName() == "CDOTA_PlayerResource" {
			for i := 0; i < 10; i++ {
				fmt.Println(reflect.TypeOf(e.Get("m_vecPlayerTeamData.000" + strconv.Itoa(i) + ".m_nSelectedHeroID")))
				heroplayers[i].Hero_ID = e.Get("m_vecPlayerTeamData.000" + strconv.Itoa(i) + ".m_nSelectedHeroID").(int32)
				intToString := e.Get("m_vecPlayerData.000" + strconv.Itoa(i) + ".m_iPlayerSteamID").(int32)
				heroplayers[i].Player_ID = fmt.Sprint(intToString)
			}
		}
		return nil
	})
	p.Start()
	for k := range glyphs {
		for l := range heroplayers {
			if fmt.Sprint(glyphs[k].User_steamID) == fmt.Sprint(heroplayers[l].Player_ID) {
				glyphs[k].HeroID = uint64(heroplayers[l].Hero_ID)
			}
		}
	}
	file, _ := json.MarshalIndent(glyphs, "", " ")
	if err != nil {
		return nil, err
	}
	write_to := "parsed_matches/" + match_id + ".json"
	_ = ioutil.WriteFile(write_to, file, 0644)

	return glyphs, nil
}
