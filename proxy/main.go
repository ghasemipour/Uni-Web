package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret   = []byte(getEnv("JWT_SECRET", "super-secret-key-12345"))
	backendURL  = getEnv("BACKEND_URL", "http://legacy-backend:8080")
	targetURL   *url.URL
	reverseProxy *httputil.ReverseProxy
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func init() {
	var err error
	targetURL, err = url.Parse(backendURL)
	if err != nil {
		log.Fatalf("Invalid backend URL: %v", err)
	}
	reverseProxy = httputil.NewSingleHostReverseProxy(targetURL)
}


func loginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "user"
	}

	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Hour * 2).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"access_token":"%s","token_type":"bearer"}`, tokenString)
}


func proxyHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, `{"detail":"Missing or invalid Authorization header"}`, http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, `{"detail":"Invalid or expired JWT token"}`, http.StatusUnauthorized)
		return
	}

	
	r.Host = targetURL.Host
	reverseProxy.ServeHTTP(w, r)
}

func main() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/", proxyHandler)

	port := ":80"
	log.Printf("Starting JWT Reverse Proxy on port %s, forwarding to %s ...", port, backendURL)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}