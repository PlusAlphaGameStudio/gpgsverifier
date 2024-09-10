package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type PlayerInfo struct {
	Kind               string `json:"kind"`
	PlayerId           string `json:"playerId"`
	DisplayName        string `json:"displayName"`
	AvatarImageUrl     string `json:"avatarImageUrl"`
	BannerUrlPortrait  string `json:"bannerUrlPortrait"`
	BannerUrlLandscape string `json:"bannerUrlLandscape"`
	ProfileSettings    struct {
		Kind                  string `json:"kind"`
		ProfileVisible        bool   `json:"profileVisible"`
		FriendsListVisibility string `json:"friendsListVisibility"`
	} `json:"profileSettings"`
	ExperienceInfo struct {
		Kind                       string `json:"kind"`
		CurrentExperiencePoints    string `json:"currentExperiencePoints"`
		LastLevelUpTimestampMillis string `json:"lastLevelUpTimestampMillis"`
		CurrentLevel               struct {
			Kind                string `json:"kind"`
			Level               int    `json:"level"`
			MinExperiencePoints string `json:"minExperiencePoints"`
			MaxExperiencePoints string `json:"maxExperiencePoints"`
		} `json:"currentLevel"`
		NextLevel struct {
			Kind                string `json:"kind"`
			Level               int    `json:"level"`
			MinExperiencePoints string `json:"minExperiencePoints"`
			MaxExperiencePoints string `json:"maxExperiencePoints"`
		} `json:"nextLevel"`
	} `json:"experienceInfo"`
	Title        string `json:"title"`
	GamePlayerId string `json:"gamePlayerId"`
}

func main() {
	goDotErr := godotenv.Load()
	if goDotErr != nil {
		log.Println("Error loading .env file")
	}

	http.HandleFunc("/verify", verifyHandler)

	listenAddr := ":60360"
	log.Println("LISTEN started on " + listenAddr)
	_ = http.ListenAndServe(listenAddr, nil)
}

func verifyHandler(writer http.ResponseWriter, request *http.Request) {
	authCode := ""

	for key := range request.Header {
		value := request.Header[key]
		log.Println(key, value)

		if key == "Auth-Code" {
			authCode = value[0]
		}
	}

	var playerInfo *PlayerInfo
	if authCode == "DummyAuthCode" {
		playerInfo = &PlayerInfo{
			PlayerId: "Dummy",
			AvatarImageUrl: "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png",
			BannerUrlLandscape: "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png",
		}
	} else {

		accessToken, err := exchangeAuthCodeToAccessToken(authCode)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		playerInfo, err = getMe(accessToken)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	writer.Header().Set("Player-Id", playerInfo.PlayerId)
	writer.Header().Set("Avatar-Image-Url", playerInfo.AvatarImageUrl)
	writer.Header().Set("Banner-Url-Landscape", playerInfo.BannerUrlLandscape)
	_, _ = writer.Write([]byte("OK"))
}

func exchangeAuthCodeToAccessToken(authCode string) (string, error) {
	// Google OAuth 2.0 클라이언트 ID 및 시크릿
	clientID := os.Getenv("GPGS_VERIFIER_CLIENT_ID")
	clientSecret := os.Getenv("GPGS_VERIFIER_CLIENT_SECRET")
	redirectURI := ""

	// POST 요청에 필요한 데이터 설정
	data := url.Values{}
	data.Set("code", authCode)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	// POST 요청 생성
	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// HTTP 클라이언트로 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
		return "", err
	}

	log.Println(string(body))

	// 응답을 JSON 형식으로 파싱
	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		log.Fatalf("Failed to parse response body: %v", err)
		return "", err
	}

	if tokenResponse["access_token"] != nil {
		// 액세스 토큰 출력
		log.Printf("Access Token: %s\n", tokenResponse["access_token"])
		log.Printf("Refresh Token: %s\n", tokenResponse["refresh_token"])
		log.Printf("Token Type: %s\n", tokenResponse["token_type"])
		log.Printf("Expires In: %f seconds\n", tokenResponse["expires_in"])

		return tokenResponse["access_token"].(string), nil
	}

	return "", errors.New("empty access token")
}

func getMe(accessToken string) (*PlayerInfo, error) {

	// Google Play Games API 엔드포인트
	apiURL := "https://games.googleapis.com/games/v1/players/me"

	// HTTP GET 요청 생성
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
		return nil, err
	}

	// Authorization 헤더에 Bearer 토큰 추가
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// HTTP 클라이언트로 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
		return nil, err
	}

	log.Println(string(body))

	// 응답을 JSON 형식으로 파싱
	var playerInfo PlayerInfo
	if err := json.Unmarshal(body, &playerInfo); err != nil {
		log.Fatalf("Failed to parse response body: %v", err)
		return nil, err
	}

	return &playerInfo, nil
}