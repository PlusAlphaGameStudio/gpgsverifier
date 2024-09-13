package main

import (
	"github.com/gasbank/gpgsverifier/verify"
	"github.com/joho/godotenv"
	"log"
	"net/http"
)

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

	playerInfo, err := verify.Verify(authCode)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.Header().Set("Player-Id", playerInfo.PlayerId)
	writer.Header().Set("Avatar-Image-Url", playerInfo.AvatarImageUrl)
	writer.Header().Set("Banner-Url-Landscape", playerInfo.BannerUrlLandscape)
	_, _ = writer.Write([]byte("OK"))
}

