package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"

	"github.com/labstack/echo/v4"
)

func TestRegisterRouteSkipsAdminRoutesInPureGatewayMode(t *testing.T) {
	viper.Reset()
	viper.Set("gateway_pure_mode", true)

	e := echo.New()
	RegisterRoute(e)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/auth/init-password", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when admin routes are disabled", rec.Code)
	}
}

func TestRegisterRouteKeepsPaymentRoutesInPureGatewayMode(t *testing.T) {
	viper.Reset()
	viper.Set("gateway_pure_mode", true)

	e := echo.New()
	RegisterRoute(e)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("root route missing in pure gateway mode: status=%d", rec.Code)
	}
}
