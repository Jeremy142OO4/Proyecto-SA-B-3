package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func baseMonitoreo(t *testing.T) *pgxpool.Pool {
	t.Helper()
	address := os.Getenv("TEST_DATABASE_URL")
	if address == "" {
		t.Skip("isolated TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	schema := "monitor_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(address)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); admin.Close() })
	files, _ := filepath.Glob("../database/migrations/*.up.sql")
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(body)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	return pool
}
func cuentaMonitoreo(t *testing.T, r *RepositorioCuentasPostgres) *models.Cuenta {
	t.Helper()
	now := time.Now().UTC()
	c := &models.Cuenta{IDCuenta: uuid.New(), IDCliente: uuid.New(), NumeroCuenta: uuid.NewString()[:18], TipoCuenta: models.TipoCuentaAhorro, SaldoCentavos: 100000, Moneda: "GTQ", Estado: models.EstadoCuentaActiva, FechaCreacion: now, FechaActualizacion: now, Version: 1}
	if err := r.Crear(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return c
}
func solicitudMonitoreo(id uuid.UUID, tipo models.TipoMovimiento) SolicitudMovimientoCuenta {
	event := events.EventoCuentaAcreditada
	if tipo == models.TipoMovimientoDebito {
		event = events.EventoCuentaDebitada
	}
	if tipo == models.TipoMovimientoCompensacion {
		event = events.EventoCuentaCompensada
	}
	return SolicitudMovimientoCuenta{IDMensaje: uuid.New(), IDCorrelacion: uuid.New(), IDCuenta: id, IDOperacion: uuid.New(), TipoMovimiento: tipo, MontoCentavos: 100, NombreConsumidor: "account-test", TipoEventoExitoso: event}
}
func adminMonitoreo(t *testing.T, r *RepositorioCuentasPostgres, tipo string, p any) events.SobreMensaje {
	t.Helper()
	body, _ := json.Marshal(p)
	m := events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New(), Tipo: tipo, Contenido: body}
	if err := r.EjecutarAdministracion(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	return m
}
func contarMonitoreo(t *testing.T, pool *pgxpool.Pool, query string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func TestMonitoreoUmbralVentanaDuplicadosYBloqueo(t *testing.T) {
	ctx := context.Background()
	pool := baseMonitoreo(t)
	r := NuevoRepositorioCuentasPostgres(pool, models.ConfiguracionMonitoreo{MaxOperaciones: 2, VentanaMinutos: 5})
	c := cuentaMonitoreo(t, r)
	debit := solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoDebito)
	for _, s := range []SolicitudMovimientoCuenta{debit, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)} {
		if _, err := r.ProcesarMovimiento(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 0 {
		t.Fatal("N operations must not alert")
	}
	third := solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)
	if _, err := r.ProcesarMovimiento(ctx, third); err != nil {
		t.Fatal(err)
	}
	if result, err := r.ProcesarMovimiento(ctx, third); err != nil || !result.Duplicado {
		t.Fatal("retry was not deduplicated")
	}
	if _, err := r.ProcesarMovimiento(ctx, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)); err != nil {
		t.Fatal(err)
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 1 {
		t.Fatal("expected one anomaly within cooldown")
	}
	var event AnomaliaCuenta
	var payload []byte
	var correlation uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT contenido,id_correlacion FROM mensajes_salida WHERE tipo_evento=$1`, EventoAccountAnomalyDetected).Scan(&payload, &correlation); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.Operaciones != 3 || event.IDCliente != c.IDCliente || event.IDCuenta != c.IDCuenta || event.Nivel != "WARNING" || correlation != third.IDCorrelacion {
		t.Fatal("invalid anomaly event")
	}
	if _, err := pool.Exec(ctx, `UPDATE movimientos_cuenta SET fecha_creacion=clock_timestamp()-interval '6 minutes';UPDATE anomalias_cuenta SET detectada_en=clock_timestamp()-interval '6 minutes'`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := r.ProcesarMovimiento(ctx, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)); err != nil {
			t.Fatal(err)
		}
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 2 {
		t.Fatal("sliding window did not expire")
	}
	block := adminMonitoreo(t, r, "cuenta.bloqueo.solicitado", map[string]any{"idCuenta": c.IDCuenta, "idAdministrador": uuid.New(), "motivo": "Revisar frecuencia inusual"})
	if err := r.EjecutarAdministracion(ctx, block); err != nil {
		t.Fatal(err)
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM mensajes_salida WHERE tipo_evento='cuenta.bloqueada'`) != 1 {
		t.Fatal("duplicate block event")
	}
	if _, err := r.ProcesarMovimiento(ctx, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)); !errors.Is(err, ErrCuentaNoActiva) {
		t.Fatalf("blocked account accepted credit: %v", err)
	}
	refund := solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCompensacion)
	refund.IDOperacion = debit.IDOperacion
	if _, err := r.ProcesarMovimiento(ctx, refund); err != nil {
		t.Fatalf("blocked account must allow compensation: %v", err)
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 2 {
		t.Fatal("compensation counted as anomaly")
	}
}
func TestMonitoreoConfiguracionPersistidaYReintento(t *testing.T) {
	ctx := context.Background()
	pool := baseMonitoreo(t)
	r := NuevoRepositorioCuentasPostgres(pool)
	adminID := uuid.New()
	m := adminMonitoreo(t, r, "cuenta.monitoreo.actualizacion.solicitada", map[string]any{"maxOperaciones": 1, "ventanaMinutos": 1, "idAdministrador": adminID})
	if err := r.EjecutarAdministracion(ctx, m); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := pool.QueryRow(ctx, `SELECT version FROM configuracion_monitoreo_cuentas`).Scan(&version); err != nil || version != 1 {
		t.Fatal("configuration replayed")
	}
	r = NuevoRepositorioCuentasPostgres(pool)
	c := cuentaMonitoreo(t, r)
	for i := 0; i < 2; i++ {
		if _, err := r.ProcesarMovimiento(ctx, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)); err != nil {
			t.Fatal(err)
		}
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 1 {
		t.Fatal("runtime config not applied after restart")
	}
	adminMonitoreo(t, r, "cuenta.monitoreo.actualizacion.solicitada", map[string]any{"maxOperaciones": 1, "ventanaMinutos": 1, "idAdministrador": adminID})
	if _, err := r.ProcesarMovimiento(ctx, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)); err != nil {
		t.Fatal(err)
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 1 {
		t.Fatal("saving identical limits must not bypass alert cooldown")
	}
	adminMonitoreo(t, r, "cuenta.monitoreo.actualizacion.solicitada", map[string]any{"maxOperaciones": 0, "ventanaMinutos": 0, "idAdministrador": adminID})
	if contarMonitoreo(t, pool, `SELECT max_operaciones FROM configuracion_monitoreo_cuentas`) != 1 {
		t.Fatal("invalid config persisted")
	}
}
func TestMonitoreoRollbackNoPublicaNiCuentaMovimiento(t *testing.T) {
	ctx := context.Background()
	pool := baseMonitoreo(t)
	r := NuevoRepositorioCuentasPostgres(pool, models.ConfiguracionMonitoreo{MaxOperaciones: 1, VentanaMinutos: 5})
	c := cuentaMonitoreo(t, r)
	if _, err := r.ProcesarMovimiento(ctx, solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, `CREATE FUNCTION fail_anomaly() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.tipo_evento='AccountAnomalyDetected' THEN RAISE EXCEPTION 'injected test failure'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER fail_anomaly BEFORE INSERT ON mensajes_salida FOR EACH ROW EXECUTE FUNCTION fail_anomaly()`)
	if err != nil {
		t.Fatal(err)
	}
	s := solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)
	if _, err = r.ProcesarMovimiento(ctx, s); err == nil {
		t.Fatal("expected injected failure")
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM movimientos_cuenta`) != 1 || contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 0 {
		t.Fatal("movement or anomaly survived rollback")
	}
	if _, err = pool.Exec(ctx, `DROP TRIGGER fail_anomaly ON mensajes_salida`); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ProcesarMovimiento(ctx, s); err != nil {
		t.Fatal(err)
	}
}
func TestMonitoreoConcurrenteEntreInstancias(t *testing.T) {
	ctx := context.Background()
	pool := baseMonitoreo(t)
	base := models.ConfiguracionMonitoreo{MaxOperaciones: 2, VentanaMinutos: 5}
	r := NuevoRepositorioCuentasPostgres(pool, base)
	c := cuentaMonitoreo(t, r)
	var group sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			other := NuevoRepositorioCuentasPostgres(pool, base)
			s := solicitudMonitoreo(c.IDCuenta, models.TipoMovimientoCredito)
			var err error
			for attempt := 0; attempt < 30; attempt++ {
				_, err = other.ProcesarMovimiento(ctx, s)
				var p *pgconn.PgError
				if !errors.As(err, &p) || p.Code != "40001" {
					break
				}
				time.Sleep(time.Millisecond * 5)
			}
			results <- err
		}()
	}
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if contarMonitoreo(t, pool, `SELECT count(*) FROM movimientos_cuenta`) != 8 || contarMonitoreo(t, pool, `SELECT count(*) FROM anomalias_cuenta`) != 1 {
		t.Fatal("concurrent detector duplicated or lost operations")
	}
}
