package main

import (
	"math"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/samber/lo"
)

//go:embed README.base.md
var fs embed.FS

func getPokemonNumber() int {
	// first four gens, because they are the best
	return rand.IntN(494)
}

// Sprites is a struct that contains the URLs for the different sprites of a pokemon
type Sprites struct {
	FrontDefault     string `json:"front_default"`
	FrontFemale      string `json:"front_female"`
	FrontShiny       string `json:"front_shiny"`
	FrontShinyFemale string `json:"front_shiny_female"`
}

type GameSprites struct {
	Sprites
	Animated Sprites `json:"animated"`
}

type Generation struct {
	BlackWhite GameSprites `json:"black-white"`
}

type Versions struct {
	// only grabbing fifth gen as it has gifs
	Fifth Generation `json:"generation-v"`
}

type TopLevelSprites struct {
	// default sprites
	Sprites
	// Various versions of the sprite
	Versions Versions `json:"versions"`
}

// Pokemon is a struct that contains the name and sprites of a pokemon
type Pokemon struct {
	Name    string          `json:"name"`
	Sprites TopLevelSprites `json:"sprites"`
}

func (p Pokemon) hasGenderedForm() bool {
	return p.Sprites.FrontFemale != "" && p.Sprites.FrontShinyFemale != ""
}

func (p Pokemon) getSprite(male bool, shiny bool) (uri string) {
	if shiny && male {
		uri = lo.CoalesceOrEmpty(p.Sprites.Versions.Fifth.BlackWhite.Animated.FrontShiny, p.Sprites.FrontShiny)
	} else if male && !shiny {
		uri = lo.CoalesceOrEmpty(p.Sprites.Versions.Fifth.BlackWhite.Animated.FrontDefault, p.Sprites.FrontDefault)
	} else if !male && shiny {
		uri = lo.CoalesceOrEmpty(p.Sprites.Versions.Fifth.BlackWhite.Animated.FrontShinyFemale, p.Sprites.FrontShinyFemale)
	} else {
		uri = lo.CoalesceOrEmpty(p.Sprites.Versions.Fifth.BlackWhite.Animated.FrontFemale, p.Sprites.FrontFemale)
	}
	return uri
}

func getPokemon(number int) (*Pokemon, error) {
	requestURL := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%d", number)
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting pokemon: %s", resp.Status)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	var pokemon Pokemon
	err = json.Unmarshal(bodyBytes, &pokemon)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	return &pokemon, nil
}

func getSprite(pokemon *Pokemon, shiny bool) string {
	male := true
	if pokemon.hasGenderedForm() {
		male = rand.Float32() < 0.5
	}
	return pokemon.getSprite(male, shiny)
}

type PlanetaryKIndex struct {
	Index float64 `json:"estimated_kp"`
}

func getPlanetaryKIndex() ([]PlanetaryKIndex, error) {
	requestURL := fmt.Sprintf("https://services.swpc.noaa.gov/json/planetary_k_index_1m.json")
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting planetary k indexes: %s", resp.Status)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	indexes := make([]PlanetaryKIndex, 0)
	err = json.Unmarshal(bodyBytes, &indexes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	return indexes, nil
}

func averageIndex(indexes []PlanetaryKIndex) float64 {
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Index > indexes[j].Index
	})
	result := lo.SumBy(indexes[:10], func(index PlanetaryKIndex) float64 {
		return index.Index
	})
	return result / 10
}

func getPShiny(index float64) float64 {
	// https://bulbapedia.bulbagarden.net/wiki/Shiny_Pok%C3%A9mon
	// return rand.IntN(65536) < 16
	// if index is high, f will be near 1 and thus it will likely be shiny
	// if index is low, f will be near 0 and thus will not be shiny
	return 1 / (1 + math.Pow(math.E, 10*(5-index)))
}

func isShiny() (bool, error) {
	indexes, err := getPlanetaryKIndex()
	if err != nil {
		log.Fatal(err)
	}
	avg := averageIndex(indexes)
	prob := getPShiny(avg)
	return rand.Float64() < prob, nil
}

func newReadme(readme string, pokemon *Pokemon, shiny bool) string {
	// update link and name
	readme = strings.Replace(string(readme), "{{link}}", getSprite(pokemon, shiny), 1)
	readme = strings.Replace(readme, "{{name}}", strings.Title(pokemon.Name), 1)
	return readme
}

// main loads root command from cmds package and executes it
func main() {
	pokemon, err := getPokemon(getPokemonNumber())
	if err != nil {
		log.Fatal(err)
	}
	shiny, err := isShiny()
	if err != nil {
		log.Fatal(err)
	}
	readme, err := fs.ReadFile("README.base.md")
	if err != nil {
		log.Fatal(err)
	}
	new := newReadme(string(readme), pokemon, shiny)
	if err := os.WriteFile("README.md", []byte(new), 0666); err != nil {
		log.Fatal(err)
	}
	return
}
