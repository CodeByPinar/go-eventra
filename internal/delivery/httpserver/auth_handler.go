package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"eventra/internal/usecase/auth"
)

type AuthHandler struct {
	authService *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	Token        string `json:"token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		CreatedAt string `json:"created_at"`
	} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeStrictJSONBody(w, r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.authService.Register(r.Context(), auth.RegisterInput{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		ClientIP:  clientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailInUse):
			respondJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		case errors.Is(err, auth.ErrUsernameInUse):
			respondJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		default:
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}

	respondJSON(w, http.StatusCreated, toAuthResponse(result))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeStrictJSONBody(w, r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.authService.Login(r.Context(), auth.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		ClientIP:  clientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, auth.ErrAccountLocked) {
			respondJSON(w, http.StatusLocked, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, toAuthResponse(result))
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeStrictJSONBody(w, r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.authService.Refresh(r.Context(), auth.RefreshInput{RefreshToken: req.RefreshToken, ClientIP: clientIP(r), UserAgent: r.UserAgent()})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, toAuthResponse(result))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := decodeStrictJSONBody(w, r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	err := h.authService.Logout(r.Context(), auth.LogoutInput{RefreshToken: req.RefreshToken, ClientIP: clientIP(r), UserAgent: r.UserAgent()})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := getAuthClaims(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"user_id": claims.UserID,
		"email":   claims.Email,
	})
}

func toAuthResponse(result auth.AuthResult) authResponse {
	res := authResponse{
		Token:        result.Token,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}
	res.User.ID = result.User.ID.String()
	res.User.Username = result.User.Username
	res.User.Email = result.User.Email
	res.User.CreatedAt = result.User.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	return res
}

func decodeStrictJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
		if contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
			return errors.New("unsupported content type")
		}
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON object")
	}

	return nil
}
