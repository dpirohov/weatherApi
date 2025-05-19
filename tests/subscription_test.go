package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"weatherApi/internal/provider"
	"weatherApi/internal/repository/subscription"
	"weatherApi/internal/repository/user"
	"weatherApi/internal/server/routes"
	s_service "weatherApi/internal/service/subscription"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(handler *routes.SubscriptionHandler) *gin.Engine {
	r := gin.Default()
	r.POST("/subscribe", handler.Subscribe)
	r.GET("/confirm/:token", handler.ConfirmSubscription)
	r.GET("/unsubscribe/:token", handler.Unsubscribe)
	return r
}

func TestSubscribeSuccess(t *testing.T) {
	userRepo := &user.MockUserRepository{
		FindOneOrCreateFn: func(c map[string]any, e *user.UserModel) (*user.UserModel, error) {
			e.ID = 1
			return e, nil
		},
	}

	subRepo := &subscription.MockSubscriptionRepository{
		FindOneOrNoneFn: func(query any, args ...any) (*subscription.SubscriptionModel, error) {
			return nil, nil
		},
		CreateOneFn: func(e *subscription.SubscriptionModel) error {
			return nil
		},
	}

	service := s_service.NewSubscriptionService(subRepo, userRepo, &provider.MockSMTPClient{}, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	body, _ := json.Marshal(gin.H{
		"email":     "test@example.com",
		"city":      "Kyiv",
		"frequency": "daily",
	})
	req, _ := http.NewRequest("POST", "/subscribe", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Subscription successful")
}

func TestSubscribeInvalidInput(t *testing.T) {
	service := s_service.NewSubscriptionService(nil, nil, nil, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	body, _ := json.Marshal(gin.H{
		"email": "test@example.com", // missing "city" and "frequency"
	})
	req, _ := http.NewRequest("POST", "/subscribe", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid input")
}

func TestConfirmSubscriptionSuccess(t *testing.T) {
	sub := &subscription.SubscriptionModel{
		IsConfirmed:  false,
		TokenExpires: time.Now().Add(1 * time.Hour),
	}
	subRepo := &subscription.MockSubscriptionRepository{
		FindOneOrNoneFn: func(query any, args ...any) (*subscription.SubscriptionModel, error) {
			return sub, nil
		},
		UpdateFn: func(e *subscription.SubscriptionModel) error {
			return nil
		},
	}
	service := s_service.NewSubscriptionService(subRepo, nil, nil, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/confirm/some-valid-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Subscription confirmed")
}

func TestConfirmSubscriptionInvalidToken(t *testing.T) {
	subRepo := &subscription.MockSubscriptionRepository{
		FindOneOrNoneFn: func(query any, args ...any) (*subscription.SubscriptionModel, error) {
			return nil, nil
		},
	}
	service := s_service.NewSubscriptionService(subRepo, nil, nil, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/confirm/invalid-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Token not found")
}

func TestTokenExpired(t *testing.T) {
	sub := &subscription.SubscriptionModel{
		IsConfirmed:  false,
		TokenExpires: time.Now().Add(-1 * time.Hour),
	}
	subRepo := &subscription.MockSubscriptionRepository{
		FindOneOrNoneFn: func(query any, args ...any) (*subscription.SubscriptionModel, error) {
			return sub, nil
		},
		UpdateFn: func(e *subscription.SubscriptionModel) error {
			return nil
		},
	}
	service := s_service.NewSubscriptionService(subRepo, nil, nil, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/confirm/some-valid-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestUnsubscribeSuccess(t *testing.T) {
	sub := &subscription.SubscriptionModel{
		IsConfirmed: true,
	}
	subRepo := &subscription.MockSubscriptionRepository{
		FindOneOrNoneFn: func(query any, args ...any) (*subscription.SubscriptionModel, error) {
			return sub, nil
		},
		DeleteFn: func(e *subscription.SubscriptionModel) error {
			return nil
		},
	}
	service := s_service.NewSubscriptionService(subRepo, nil, nil, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/unsubscribe/some-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Unsubscribed successfully")
}

func TestUnsubscribeSubscriptionTokenNotFound(t *testing.T) {

	subRepo := &subscription.MockSubscriptionRepository{
		FindOneOrNoneFn: func(query any, args ...any) (*subscription.SubscriptionModel, error) {
			return nil, nil
		},
	}
	service := s_service.NewSubscriptionService(subRepo, nil, nil, 60)
	handler := routes.NewSubscriptionHandler(service)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/unsubscribe/some-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Token not found")
}
