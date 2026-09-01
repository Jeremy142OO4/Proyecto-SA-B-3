package operations

import (
	"sync"
	"time"
)

type Operacion struct {
	OperationID   string    `json:"operationId"`
	CorrelationID string    `json:"correlationId"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ErrorCode     string    `json:"errorCode,omitempty"`
}

type Store struct {
	mu             sync.RWMutex
	datos          map[string]Operacion
	porCorrelacion map[string]string
}

func NuevoStore() *Store {
	return &Store{datos: map[string]Operacion{}, porCorrelacion: map[string]string{}}
}
func (s *Store) Crear(o Operacion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.datos[o.OperationID] = o
	s.porCorrelacion[o.CorrelationID] = o.OperationID
}
func (s *Store) Obtener(id string) (Operacion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.datos[id]
	return o, ok
}
func (s *Store) Actualizar(correlacion, estado, codigo string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.porCorrelacion[correlacion]
	if !ok {
		return
	}
	o := s.datos[id]
	o.Status = estado
	o.ErrorCode = codigo
	o.UpdatedAt = time.Now().UTC()
	s.datos[id] = o
}
