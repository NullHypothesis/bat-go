package subscriptions

// Code generated by gowrap. DO NOT EDIT.
// template: ../.prom-gowrap.tmpl
// gowrap: http://github.com/hexdigest/gowrap

//go:generate gowrap gen -p github.com/brave-intl/bat-go/subscriptions -i Datastore -t ../.prom-gowrap.tmpl -o instrumented_datastore.go -l ""

import (
	"time"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DatastoreWithPrometheus implements Datastore interface with all methods wrapped
// with Prometheus metrics
type DatastoreWithPrometheus struct {
	base         Datastore
	instanceName string
}

var datastoreDurationSummaryVec = promauto.NewSummaryVec(
	prometheus.SummaryOpts{
		Name:       "datastore_duration_seconds",
		Help:       "datastore runtime duration and result",
		MaxAge:     time.Minute,
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"instance_name", "method", "result"})

// NewDatastoreWithPrometheus returns an instance of the Datastore decorated with prometheus summary metric
func NewDatastoreWithPrometheus(base Datastore, instanceName string) DatastoreWithPrometheus {
	return DatastoreWithPrometheus{
		base:         base,
		instanceName: instanceName,
	}
}

// Migrate implements Datastore
func (_d DatastoreWithPrometheus) Migrate(p1 ...uint) (err error) {
	_since := time.Now()
	defer func() {
		result := "ok"
		if err != nil {
			result = "error"
		}

		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "Migrate", result).Observe(time.Since(_since).Seconds())
	}()
	return _d.base.Migrate(p1...)
}

// NewMigrate implements Datastore
func (_d DatastoreWithPrometheus) NewMigrate() (mp1 *migrate.Migrate, err error) {
	_since := time.Now()
	defer func() {
		result := "ok"
		if err != nil {
			result = "error"
		}

		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "NewMigrate", result).Observe(time.Since(_since).Seconds())
	}()
	return _d.base.NewMigrate()
}

// RawDB implements Datastore
func (_d DatastoreWithPrometheus) RawDB() (dp1 *sqlx.DB) {
	_since := time.Now()
	defer func() {
		result := "ok"
		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "RawDB", result).Observe(time.Since(_since).Seconds())
	}()
	return _d.base.RawDB()
}

// RollbackTx implements Datastore
func (_d DatastoreWithPrometheus) RollbackTx(tx *sqlx.Tx) {
	_since := time.Now()
	defer func() {
		result := "ok"
		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "RollbackTx", result).Observe(time.Since(_since).Seconds())
	}()
	_d.base.RollbackTx(tx)
	return
}

// RollbackTxAndHandle implements Datastore
func (_d DatastoreWithPrometheus) RollbackTxAndHandle(tx *sqlx.Tx) (err error) {
	_since := time.Now()
	defer func() {
		result := "ok"
		if err != nil {
			result = "error"
		}

		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "RollbackTxAndHandle", result).Observe(time.Since(_since).Seconds())
	}()
	return _d.base.RollbackTxAndHandle(tx)
}

// createRoom implements Datastore
func (_d DatastoreWithPrometheus) createRoom(r Room) (err error) {
	_since := time.Now()
	defer func() {
		result := "ok"
		if err != nil {
			result = "error"
		}

		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "createRoom", result).Observe(time.Since(_since).Seconds())
	}()
	return _d.base.createRoom(r)
}

// increaseMau implements Datastore
func (_d DatastoreWithPrometheus) increaseMau() {
	_since := time.Now()
	defer func() {
		result := "ok"
		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "increaseMau", result).Observe(time.Since(_since).Seconds())
	}()
	_d.base.increaseMau()
	return
}

// joinRoom implements Datastore
func (_d DatastoreWithPrometheus) joinRoom(r Room) (err error) {
	_since := time.Now()
	defer func() {
		result := "ok"
		if err != nil {
			result = "error"
		}

		datastoreDurationSummaryVec.WithLabelValues(_d.instanceName, "joinRoom", result).Observe(time.Since(_since).Seconds())
	}()
	return _d.base.joinRoom(r)
}
