package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
)

//go:embed README.base.md
var fs embed.FS

func getPokemonNumber() int {
	// first four gens, because they are the best
	return rand.IntN(494)
}

func isShiny() bool {
	// https://bulbapedia.bulbagarden.net/wiki/Shiny_Pok%C3%A9mon
	return rand.IntN(65536) < 16
}

// Sprites is a struct that contains the URLs for the different sprites of a pokemon
type Sprites struct {
	FrontDefault     string `json:"front_default"`
	FrontFemale      string `json:"front_female"`
	FrontShiny       string `json:"front_shiny"`
	FrontShinyFemale string `json:"front_shiny_female"`
}

func (s Sprites) hasGenderedForm() bool {
	return s.FrontFemale != "" && s.FrontShinyFemale != ""
}

func (s Sprites) getSprite(male bool, shiny bool) (uri string) {
	if shiny {
		switch male {
		case true:
			uri = s.FrontShiny
		default:
			uri = s.FrontShinyFemale
		}
	} else {
		switch male {
		case true:
			uri = s.FrontDefault
		default:
			uri = s.FrontFemale
		}
	}
	return uri
}

// Pokemon is a struct that contains the name and sprites of a pokemon
type Pokemon struct {
	Name    string  `json:"name"`
	Sprites Sprites `json:"sprites"`
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

func getSprite(pokemon *Pokemon) string {
	male := true
	if pokemon.Sprites.hasGenderedForm() {
		male = rand.Float32() < 0.5
	}
	return pokemon.Sprites.getSprite(male, isShiny())
}

func getRandomEncounter() (string, error) {
	pokemon, err := getPokemon(getPokemonNumber())
	if err != nil {
		return "", fmt.Errorf("error getting pokemon: %w", err)
	}
	return getSprite(pokemon), nil
}

// main loads root command from cmds package and executes it
func main() {
	encounter, err := getRandomEncounter()
	if err != nil {
		log.Fatal(err)
	}
	readme, err := fs.ReadFile("README.base.md")
	if err != nil {
		log.Fatal(err)
	}
	newREADME := strings.Replace(string(readme), "{{pokemon}}", encounter, 1)
	if err := os.WriteFile("README.md", []byte(newREADME), 0666); err != nil {
		log.Fatal(err)
	}
	return
}
