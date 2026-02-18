package main

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
)

//go:embed www/*
var staticFiles embed.FS

var usernamePattern = regexp.MustCompile(`^[a-z0-9._=\-/]+$`)

func main() {
	inviteCode := os.Getenv("INVITE_CODE")
	if inviteCode == "" {
		log.Fatal("INVITE_CODE is not set")
	}

	homeserverAddr := os.Getenv("HOMESERVER_URL")
	if homeserverAddr == "" {
		homeserverAddr = "http://localhost:6167"
	}
	homeserverURL, err := url.Parse(homeserverAddr)
	if err != nil {
		log.Fatalf("invalid HOMESERVER_URL %q: %v", homeserverAddr, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(homeserverURL)

	wwwFS, err := fs.Sub(staticFiles, "www")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(wwwFS))

	mux := http.NewServeMux()

	mux.Handle("/register/", http.StripPrefix("/register/", fileServer))
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/register/", http.StatusMovedPermanently)
	})

	mux.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleRegistration(w, r, inviteCode, homeserverURL.String())
	})

	mux.Handle("/", proxy)

	log.Println("registration proxy listening on :8008")
	log.Fatal(http.ListenAndServe(":8008", mux))
}

type registrationRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
}

type uiaResponse struct {
	Session string `json:"session"`
}

func handleRegistration(w http.ResponseWriter, r *http.Request, inviteCode, homeserverBase string) {
	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.InviteCode != inviteCode {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid invite code"})
		return
	}

	req.Username = strings.TrimSpace(strings.ToLower(req.Username))

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}

	if len(req.Username) > 255 || !usernamePattern.MatchString(req.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "username must contain only lowercase letters, numbers, dots, underscores, hyphens, equals, and slashes",
		})
		return
	}

	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	registerURL := homeserverBase + "/_matrix/client/v3/register"

	// Step 1: Initiate registration to get a UIA session
	initBody := map[string]any{
		"username": req.Username,
		"password": req.Password,
	}
	initJSON, _ := json.Marshal(initBody)
	initResp, err := http.Post(registerURL, "application/json", strings.NewReader(string(initJSON)))
	if err != nil {
		log.Printf("failed to initiate registration: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to contact homeserver"})
		return
	}
	defer initResp.Body.Close()
	initRespBody, _ := io.ReadAll(initResp.Body)

	// Registration succeeded without UIA (open registration)
	if initResp.StatusCode == http.StatusOK {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "account created successfully"})
		return
	}

	if initResp.StatusCode != http.StatusUnauthorized {
		var errResp map[string]any
		if json.Unmarshal(initRespBody, &errResp) == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				writeJSON(w, initResp.StatusCode, map[string]string{"error": errMsg})
				return
			}
		}
		writeJSON(w, initResp.StatusCode, map[string]string{"error": "registration failed"})
		return
	}

	// Parse session ID from UIA 401 response
	var uia uiaResponse
	if err := json.Unmarshal(initRespBody, &uia); err != nil || uia.Session == "" {
		log.Printf("failed to parse UIA response: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "unexpected response from homeserver"})
		return
	}

	// Step 2: Complete registration with the token
	regBody := map[string]any{
		"username": req.Username,
		"password": req.Password,
		"auth": map[string]any{
			"type":    "m.login.registration_token",
			"token":   inviteCode,
			"session": uia.Session,
		},
	}
	regJSON, _ := json.Marshal(regBody)
	regResp, err := http.Post(registerURL, "application/json", strings.NewReader(string(regJSON)))
	if err != nil {
		log.Printf("failed to complete registration: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to contact homeserver"})
		return
	}
	defer regResp.Body.Close()

	body, _ := io.ReadAll(regResp.Body)

	if regResp.StatusCode != http.StatusOK {
		var errResp map[string]any
		if json.Unmarshal(body, &errResp) == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				writeJSON(w, regResp.StatusCode, map[string]string{"error": errMsg})
				return
			}
		}
		writeJSON(w, regResp.StatusCode, map[string]string{"error": "registration failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "account created successfully"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
