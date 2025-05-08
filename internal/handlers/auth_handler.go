package handlers

import (
	"encoding/json"
	"net/http"
	"open-library-explorer/internal/utils"
)

type AuthHandler struct {
	ConfigCreds struct {
		UserId       string
		Username     string
		UserPassword string
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	expectedPassword := a.ConfigCreds.UserPassword
	expectedUsername := a.ConfigCreds.Username
	if expectedPassword != req.Password || expectedUsername != req.Username {
		utils.JSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, _ := utils.GenerateJWT(a.ConfigCreds.UserId)

	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
