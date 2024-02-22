package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var templates = template.Must(template.ParseFiles("pages/index.html", "pages/track.html", "pages/tracknew.html"))
var dataDir = "./data"
var allTracks []Track
var config Config

type Config struct {
	Port int `json:"port"`

	AdminUsername string `json:"admin-username"`
	AdminPassword string `json:"admin-password"`
}

type Track struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ReleaseDate int    `json:"release-date"` // Unix epoch
}

type HomePage struct {
	Tracks   []Track
	LoggedIn bool
}

type TrackPage struct {
	Track    Track
	LoggedIn bool
}

type TrackNewPage struct {
	LoggedIn bool
}

func isLoggedIn(username, password string, ok bool) bool {
	if !ok {
		return false
	}

	if subtle.ConstantTimeCompare([]byte(config.AdminUsername), []byte(username)) == 0 || subtle.ConstantTimeCompare([]byte(config.AdminPassword), []byte(password)) == 0 {
		return false
	}

	return true
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

	bytes, err := io.ReadAll(infoJSONFile)
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

	err = os.WriteFile(path+"/info.json", bytes, 0644)
	return err
}

func getAllTracks() ([]Track, error) {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	var tracks []Track

	for _, dirEntry := range files {
		track, err := getTrackFromPath(dataDir + "/" + dirEntry.Name()) // TODO: Find an absolute path function for this
		if err != nil {
			continue
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

func homePageHandler(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	templates.ExecuteTemplate(w, "index.html", HomePage{Tracks: allTracks, LoggedIn: isLoggedIn(username, password, ok)})
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

	username, password, ok := r.BasicAuth()
	templates.ExecuteTemplate(w, "track.html", TrackPage{Track: track, LoggedIn: isLoggedIn(username, password, ok)})
}

func trackSourceHandler(w http.ResponseWriter, r *http.Request) {
	trackName := strings.TrimPrefix(r.URL.Path, "/tracksource/")

	if !isTrackNameAllowed(trackName) {
		http.ServeFile(w, r, "pages/404.html")
		return
	}

	http.ServeFile(w, r, dataDir+"/"+trackName+"/audio/track.mp3")
}

func trackImageHandler(w http.ResponseWriter, r *http.Request) {
	trackName := strings.TrimPrefix(r.URL.Path, "/trackimage/")
	if !isTrackNameAllowed(trackName) {
		http.ServeFile(w, r, "pages/404.html")
		return
	}

	http.ServeFile(w, r, dataDir+"/"+trackName+"/image/cover.png")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !isLoggedIn(username, password, ok) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Access\"")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, "/", 303)
}

func trackNewHandler(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	loggedIn := isLoggedIn(username, password, ok)
	if !loggedIn {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Access\"")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		fmt.Println(r)
	} else {
		templates.ExecuteTemplate(w, "tracknew.html", TrackNewPage{LoggedIn: loggedIn})
	}
}

func main() {
	configJSONFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println("Failed to open \"config.json\", error: " + err.Error())
		os.Exit(1)
	}

	defer configJSONFile.Close()

	bytes, err := io.ReadAll(configJSONFile)
	if err != nil {
		fmt.Println("Failed to read \"config.json\", error: " + err.Error())
		os.Exit(2)
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		fmt.Println("Failed to parse \"config.json\", error: " + err.Error())
		os.Exit(3)
	}

	fmt.Println("Config:")
	fmt.Println(config)

	allTracks, err = getAllTracks()
	if err != nil {
		fmt.Println("Failed to read track data, error: " + err.Error())
		os.Exit(4)
	}

	fmt.Println(allTracks)

	http.HandleFunc("/", homePageHandler)
	http.HandleFunc("/track/", trackHandler)
	http.HandleFunc("/tracknew", trackNewHandler)
	http.HandleFunc("/tracksource/", trackSourceHandler)
	http.HandleFunc("/trackimage/", trackImageHandler)
	http.HandleFunc("/login", loginHandler)

	http.HandleFunc("/main.css", mainCSSHandler)
	http.HandleFunc("/img/icon.png", iconHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
