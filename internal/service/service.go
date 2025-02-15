package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/basedalex/merch-shop/internal/auth"
	"github.com/basedalex/merch-shop/internal/db"
	api "github.com/basedalex/merch-shop/internal/swagger"
)

type MyService struct {
	db db.Repository
}

// (POST /api/auth)
func (s *MyService) PostApiAuth(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)
	
		return
	}

	defer r.Body.Close()

	var authRequest api.AuthRequest

	err = json.Unmarshal(body, &authRequest)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)

		return
	}


	// if user exists and password is right give back token
	exists, err := s.db.Authenticate(r.Context(), authRequest); 
	if err != nil {
		log.Warn(err)
	}
	if err != nil && exists {
		writeErrResponse(w, fmt.Errorf("error: credentials are incorrect %w", err), http.StatusUnauthorized)

		return 
	}
	log.Info("exists", exists)
	
	if exists {
		token, err := auth.CreateToken(authRequest.Username)
		log.Info("Generated token ", token)
		if err != nil {
			writeErrResponse(w, err, http.StatusInternalServerError)
		
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, token)
		return
	}

	// if user doesn't exist create one

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(authRequest.Password), bcrypt.DefaultCost)
    if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)

		return
    }
	authRequest.Password = string(hashedPassword)
	fmt.Println(authRequest.Password)

	if err = s.db.CreateEmployee(r.Context(), authRequest); err != nil {
		writeErrResponse(w, fmt.Errorf("could not create new employee %w", err), http.StatusInternalServerError)
	
		return
	}

	token, err := auth.CreateToken(authRequest.Username)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, token)
}

// (GET /api/buy/{item})
func (s *MyService) GetApiBuyItem(w http.ResponseWriter, r *http.Request, item string) {
	fmt.Println("buying item")
	tokenString := r.Header.Get("Authorization")
	token := strings.TrimPrefix(tokenString, "Bearer ")
	username, err := auth.ExtractUsername(token)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)
	
		return
	}

	employeeID, err := s.db.GetEmployeeID(r.Context(), username)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}

	if err = s.db.BuyItem(r.Context(), employeeID, item); err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}
	writeOkResponse(w, http.StatusOK, nil)
}



// (GET /api/info)
func (s *MyService) GetApiInfo(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	token := strings.TrimPrefix(tokenString, "Bearer ")

	username, err := auth.ExtractUsername(token)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)
		return
	}

	employeeID, err := s.db.GetEmployeeID(r.Context(), username)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}
	
	infoResponse, err := s.db.GetEmployeeInfo(r.Context(), employeeID)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}

	writeOkResponse(w, http.StatusOK, infoResponse)
}

// (POST /api/sendCoin)
func (s *MyService) PostApiSendCoin(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)
	
		return
	}

	defer r.Body.Close()

	var sendCoinRequest api.SendCoinRequest

	err = json.Unmarshal(body, &sendCoinRequest)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)

		return
	}

	tokenString := r.Header.Get("Authorization")
	
	token := strings.TrimPrefix(tokenString, "Bearer ")

	
	username, err := auth.ExtractUsername(token)
	if err != nil {
		writeErrResponse(w, err, http.StatusBadRequest)
		return
	}

	senderID, err := s.db.GetEmployeeID(r.Context(), username)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}
	recieverID, err := s.db.GetEmployeeID(r.Context(), sendCoinRequest.ToUser)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}

	err = s.db.TransferCoins(r.Context(), senderID, recieverID, sendCoinRequest.Amount)
	if err != nil {
		writeErrResponse(w, err, http.StatusInternalServerError)
	
		return
	}
	writeOkResponse(w, http.StatusAccepted, nil)
}

func NewService(db db.Repository) *MyService {
	return &MyService{
		db: db,
	}
}

type HTTPResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeOkResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(HTTPResponse{Data: data})
	if err != nil {
		log.Warn(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func writeErrResponse(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	log.Warn(err)

	jsonErr := json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})
	if jsonErr != nil {
		log.Warn(jsonErr)
	}
}
