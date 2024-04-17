package health

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHeathRoute(t *testing.T) {
	// Define a structure for specifying input and output data
	// of a single test case
	tests := []struct {
		description  string // description of the test case
		route        string // route path to test
		expectedCode int    // expected HTTP status code
		expectedBody string // expected body
	}{
		// First test case
		{
			description:  "get HTTP status 200",
			route:        "/health",
			expectedCode: 200,
			expectedBody: "{\"status\":\"OK\"}",
		},
		// Second test case
		{
			description:  "get HTTP status 404, when route is not exists",
			route:        "/not-found",
			expectedCode: 404,
			expectedBody: "{\"error\":\"page not found\"}",
		},
	}

	// Create a new instance of the Handler struct
	h := &Handler{}

	// Create a new instance of the Fiber app
	app := fiber.New()

	// Create a new instance of the Router struct and pass it to the SetupRoutes method
	app.Route("/api/v1", func(router fiber.Router) {
		h.SetupRoutes(router)
	})

	// Handle 404 page
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(map[string]string{"error": "page not found"})
	})

	// Iterate through test single test cases
	for _, test := range tests {
		// Create a new http request with the route from the test case
		req := httptest.NewRequest("GET", "/api/v1"+test.route, nil)

		// Perform the request plain with the app,
		// the second argument is a request latency
		// (set to -1 for no latency)
		resp, err := app.Test(req, 1)
		if err != nil {
			t.Fatalf("failed to send test request: %v", err)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		// Verify, if the status code is as expected
		assert.Equalf(t, test.expectedCode, resp.StatusCode, test.description)

		// Verify, if the body is as expected
		assert.Equalf(t, test.expectedBody, string(body), test.description)
	}
}
