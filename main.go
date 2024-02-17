package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var templates = template.Must(template.ParseFiles("pages/index.html", "pages/track.html"))
var dataDir = "./data"
var allTracks []Track

type Track struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ReleaseDate int    `json:"release-date"` // Unix epoch
}

func isTrackNameAllowed(name string) bool {
	allowedChars := "abcdefghijklmnopqrstuvwxyz0123456789_-()!?,. &"
	for _, c := range name {
		if !strings.Contains(allowedChars, strings.ToLower(string(c))) {
			return false
		}
	}

	return true
}

func getTrackFromPath(path string) (Track, error) {
	var track Track

	infoJSONFile, err := os.Open(path + "/info.json")
	if err != nil {
		return track, err
	}

	defer infoJSONFile.Close()

	bytes, err := ioutil.ReadAll(infoJSONFile)
	if err != nil {
		return track, err
	}

	err = json.Unmarshal(bytes, &track)

	return track, err
}

func writeTrackToPath(track Track, path string) error {
	if !isTrackNameAllowed(track.Title) {
		return errors.New("Disallowed track title")
	}

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(track)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path+"/info.json", bytes, 0644)
	return err
}

func getAllTracks() ([]Track, error) {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	var tracks []Track

	for _, dirEntry := range files {
		fmt.Println("direntry !!! : " + dirEntry.Name())
		track, err := getTrackFromPath(dataDir + "/" + dirEntry.Name()) // TODO: Find an absolute path function for this
		if err != nil {
			continue
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

type HomePage struct {
	Tracks []Track
}

func homePageHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", HomePage{Tracks: allTracks})
}

func mainCSSHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "pages/main.css")
}

func iconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "pages/img/icon.png")
}

func trackHandler(w http.ResponseWriter, r *http.Request) {
	trackName := strings.TrimPrefix(r.URL.Path, "/track/")

	if !isTrackNameAllowed(trackName) {
		http.ServeFile(w, r, "pages/404.html")
		return
	}

	track, err := getTrackFromPath(dataDir + "/" + trackName)
	if err != nil {
		http.ServeFile(w, r, "pages/404.html")
		return
	}
	templates.ExecuteTemplate(w, "track.html", track)
}

func trackSourceHandler(w http.ResponseWriter, r *http.Request) {
	trackName := strings.TrimPrefix(r.URL.Path, "/tracksource/")

	if !isTrackNameAllowed(trackName) {
		http.ServeFile(w, r, "pages/404.html")
		return
	}

	http.ServeFile(w, r, dataDir + "/" + trackName + "/audio/track.mp3")
}

func trackImageHandler(w http.ResponseWriter, r *http.Request) {
	trackName := strings.TrimPrefix(r.URL.Path, "/trackimage/")
	if !isTrackNameAllowed(trackName) {
		http.ServeFile(w, r, "pages/404.html")
		return
	}

	http.ServeFile(w, r, dataDir + "/" + trackName + "/image/cover.png")
}

func main() {
	var err error
	allTracks, err = getAllTracks()
	if err != nil {
		os.Exit(0)
	}

	fmt.Println(allTracks)

	http.HandleFunc("/", homePageHandler)
	http.HandleFunc("/track/", trackHandler)
	http.HandleFunc("/tracksource/", trackSourceHandler)
	http.HandleFunc("/trackimage/", trackImageHandler)
	http.HandleFunc("/main.css", mainCSSHandler)

	http.HandleFunc("/img/icon.png", iconHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
