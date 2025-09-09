package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type JWTHeader struct {
	Kid string `json:"kid"`
}

type JWTPayload struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

type Metric struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n,omitempty"`
	E   string `json:"e,omitempty"`
}

func main() {
	addr := os.Getenv("ADDRESS")
	if addr == "" {
		panic(fmt.Errorf("address is empty"))
	}

	http.Handle("/reports", corsMiddleware(http.HandlerFunc(getReport)))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getReport(rw http.ResponseWriter, r *http.Request) {
	// Собирает данные из абстрактной olap db - clickhouse
	// В нее уже абстрактный dag airflow положил абстрактный данные
	_, err := verifyJWT(rw, r)
	if err != nil {
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := make([]Metric, 0)
	for i := 1; i <= 100; i++ {
		var item Metric

		item.ID = uuid.NewString()
		item.Title = fmt.Sprintf("title: %v", i)
		item.Description = fmt.Sprintf("lorem ipsum: %v", i)
		data = append(data, item)
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(data)
	rw.WriteHeader(http.StatusOK)
}

func verifyJWT(rw http.ResponseWriter, r *http.Request) (*JWTPayload, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(rw, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return nil, fmt.Errorf("missing or invalid authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		http.Error(rw, "Invalid token format", http.StatusUnauthorized)
		return nil, fmt.Errorf("invalid token format")
	}

	headerData := parts[0]
	headerData += strings.Repeat("=", (4-len(headerData)%4)%4)

	var header JWTHeader
	headerBytes, err := base64.URLEncoding.DecodeString(headerData)
	if err != nil {
		http.Error(rw, "Invalid token header", http.StatusUnauthorized)
		return nil, err
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		http.Error(rw, "Invalid token header JSON", http.StatusUnauthorized)
		return nil, err
	}

	keys, err := getPublicKey()
	if err != nil {
		http.Error(rw, "Failed to fetch keys", http.StatusInternalServerError)
		return nil, err
	}

	var key *JWK
	for _, k := range keys {
		if k.Kid == header.Kid {
			key = &k
			break
		}
	}
	if key == nil {
		http.Error(rw, "Key not found", http.StatusUnauthorized)
		return nil, fmt.Errorf("key not found")
	}

	pubKey, err := key.ToRSAPublicKey()
	if err != nil {
		http.Error(rw, "Failed to parse public key", http.StatusInternalServerError)
		return nil, err
	}

	var payload JWTPayload
	_, err = jwt.ParseWithClaims(tokenString, &payload, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		http.Error(rw, "Invalid token", http.StatusUnauthorized)
		return nil, err
	}

	return &payload, nil
}

func getPublicKey() ([]JWK, error) {
	keycloakURL := os.Getenv("API_APP_KEYCLOAK_INTERNAL_URL")
	realm := os.Getenv("API_APP_KEYCLOAK_REALM")
	if keycloakURL == "" || realm == "" {
		return nil, fmt.Errorf("environment variables not set")
	}

	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", keycloakURL, realm)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get keys: %s", resp.Status)
	}

	var result struct {
		Keys []JWK `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Keys, nil
}

func (j *JWK) ToRSAPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(j.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return nil, err
	}

	eInt := 0
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}
	return pubKey, nil
}

func (p *JWTPayload) Valid() error {
	// Можно добавить проверки срока жизни токена (exp), not before (nbf) и т.д.
	// Для простоты пока просто возвращаем nil
	return nil
}
