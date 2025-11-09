package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/config"
	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

// Server agrupa deps para la capa HTTP.
type Server struct {
	cfg             config.Config
	stockRepo       domain.StockItemRepository
	reservationRepo domain.StockReservationRepository
}

func NewServer(
	cfg config.Config,
	stockRepo domain.StockItemRepository,
	reservationRepo domain.StockReservationRepository,
) *Server {
	return &Server{
		cfg:             cfg,
		stockRepo:       stockRepo,
		reservationRepo: reservationRepo,
	}
}

// RegisterRoutes registra todas las rutas HTTP en el mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/inventory/", s.handleGetInventoryBySku)
	mux.HandleFunc("/api/reservations/", s.handleGetReservationByOrder)
	mux.HandleFunc("/swagger.json", s.handleSwaggerJson)
}

// Respuesta de health.
type healthResponse struct {
	Status string `json:"status"`
}

// Respuesta de inventario.
type inventoryResponse struct {
	Sku       string `json:"sku"`
	Available int    `json:"available"`
	Reserved  int    `json:"reserved"`
}

// Respuesta de linea de reservacion.
type reservationLineResponse struct {
	Sku      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// Respuesta de reservacion.
type reservationResponse struct {
	OrderID       uuid.UUID                 `json:"orderId"`
	UserID        uuid.UUID                 `json:"userId"`
	Status        string                    `json:"status"`
	ReservedAtUtc string                    `json:"reservedAtUtc"`
	ReleasedAtUtc *string                   `json:"releasedAtUtc,omitempty"`
	Lines         []reservationLineResponse `json:"lines"`
}

// Handler /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// Handler GET /api/inventory/{sku}
func (s *Server) handleGetInventoryBySku(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path esperado: /api/inventory/{sku}
	path := strings.TrimPrefix(r.URL.Path, "/api/inventory/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "sku is required", http.StatusBadRequest)
		return
	}
	sku := path

	ctx := r.Context()
	itemsMap, err := s.stockRepo.GetBySkus(ctx, []string{sku})
	if err != nil {
		log.Printf("GetBySkus error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	item, ok := itemsMap[sku]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	resp := inventoryResponse{
		Sku:       item.Sku,
		Available: item.Available,
		Reserved:  item.Reserved,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Handler GET /api/reservations/{orderId}
func (s *Server) handleGetReservationByOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path esperado: /api/reservations/{orderId}
	path := strings.TrimPrefix(r.URL.Path, "/api/reservations/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "orderId is required", http.StatusBadRequest)
		return
	}
	orderIDStr := path

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		http.Error(w, "orderId is invalid", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	res, err := s.reservationRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		log.Printf("GetByOrderID error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if res == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var releasedStr *string
	if res.ReleasedAtUtc != nil {
		sv := res.ReleasedAtUtc.UTC().Format("2006-01-02T15:04:05Z")
		releasedStr = &sv
	}

	lines := make([]reservationLineResponse, 0, len(res.Lines))
	for _, l := range res.Lines {
		lines = append(lines, reservationLineResponse{
			Sku:      l.Sku,
			Quantity: l.Quantity,
		})
	}

	resp := reservationResponse{
		OrderID:       res.OrderID,
		UserID:        res.UserID,
		Status:        string(res.Status),
		ReservedAtUtc: res.ReservedAtUtc.UTC().Format("2006-01-02T15:04:05Z"),
		ReleasedAtUtc: releasedStr,
		Lines:         lines,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Handler GET /swagger.json
func (s *Server) handleSwaggerJson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpec))
}

// Util para escribir JSON
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}

// Spec OpenAPI minimal en JSON para Swagger.
const openAPISpec = `{
  "openapi": "3.0.0",
  "info": {
    "title": "Inventory Service API",
    "version": "1.0.0"
  },
  "paths": {
    "/health": {
      "get": {
        "summary": "Health check",
        "responses": {
          "200": {
            "description": "Service is healthy",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/HealthResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/inventory/{sku}": {
      "get": {
        "summary": "Get inventory by sku",
        "parameters": [
          {
            "name": "sku",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Inventory found",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/InventoryResponse"
                }
              }
            }
          },
          "404": {
            "description": "Inventory not found"
          }
        }
      }
    },
    "/api/reservations/{orderId}": {
      "get": {
        "summary": "Get reservation by order id",
        "parameters": [
          {
            "name": "orderId",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string",
              "format": "uuid"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Reservation found",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ReservationResponse"
                }
              }
            }
          },
          "404": {
            "description": "Reservation not found"
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "HealthResponse": {
        "type": "object",
        "properties": {
          "status": {
            "type": "string"
          }
        }
      },
      "InventoryResponse": {
        "type": "object",
        "properties": {
          "sku": {
            "type": "string"
          },
          "available": {
            "type": "integer"
          },
          "reserved": {
            "type": "integer"
          }
        }
      },
      "ReservationLineResponse": {
        "type": "object",
        "properties": {
          "sku": {
            "type": "string"
          },
          "quantity": {
            "type": "integer"
          }
        }
      },
      "ReservationResponse": {
        "type": "object",
        "properties": {
          "orderId": {
            "type": "string",
            "format": "uuid"
          },
          "userId": {
            "type": "string",
            "format": "uuid"
          },
          "status": {
            "type": "string"
          },
          "reservedAtUtc": {
            "type": "string",
            "format": "date-time"
          },
          "releasedAtUtc": {
            "type": "string",
            "format": "date-time",
            "nullable": true
          },
          "lines": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/ReservationLineResponse"
            }
          }
        }
      }
    }
  }
}`
