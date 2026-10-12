package audit

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yuusufyan/go-common/pkg/database"
	"github.com/yuusufyan/go-common/pkg/logger"
)

// Config configures a Recorder.
type Config struct {
	// Store persists trails (e.g. NewGormStore(db)). When nil, trails are only logged.
	Store Store
	// Logger writes every trail as a structured log entry. When nil, trails are not logged.
	Logger logger.Logger
	// BufferSize is the queue capacity (default: 1000). Trails are dropped when it is full.
	BufferSize int
	// BatchSize is the max number of trails per insert (default: 100).
	BatchSize int
	// FlushInterval is the max time a trail waits in the queue (default: 1s).
	FlushInterval time.Duration
	// SaveTimeout bounds a single Store.Save call (default: 10s).
	SaveTimeout time.Duration
	// SensitiveKeys are masked in old/new values. Defaults to logger.DefaultSensitiveKeys.
	SensitiveKeys []string
}

// Recorder writes audit trails asynchronously: Record enqueues, a background
// worker saves in batches. Call Close on shutdown to flush pending trails.
type Recorder struct {
	cfg   Config
	queue chan *Trail
	done  chan struct{}

	mu     sync.RWMutex
	closed bool
}

// NewRecorder creates a Recorder and starts its background worker.
func NewRecorder(cfg Config) *Recorder {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.SaveTimeout <= 0 {
		cfg.SaveTimeout = 10 * time.Second
	}

	r := &Recorder{
		cfg:   cfg,
		queue: make(chan *Trail, cfg.BufferSize),
		done:  make(chan struct{}),
	}
	go r.run()
	return r
}

// Record fills request/trace/user info from ctx, logs the trail and enqueues it for saving.
// It never blocks: when the queue is full the trail is dropped and a warning is logged.
func (r *Recorder) Record(ctx context.Context, t *Trail) {
	if t == nil {
		return
	}
	r.fill(ctx, t)
	r.log(ctx, t)

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return
	}

	select {
	case r.queue <- t:
	default:
		if r.cfg.Logger != nil {
			r.cfg.Logger.WithCtx(ctx).WithField("audit_id", t.ID).Warn("Audit trail queue full, trail dropped")
		}
	}
}

// Close stops accepting trails and waits until pending ones are saved or ctx is done.
func (r *Recorder) Close(ctx context.Context) error {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.queue)
	}
	r.mu.Unlock()

	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Recorder) fill(ctx context.Context, t *Trail) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	t.OldValues = mask(t.OldValues, r.cfg.SensitiveKeys)
	t.NewValues = mask(t.NewValues, r.cfg.SensitiveKeys)

	if ctx == nil {
		return
	}
	if t.RequestID == "" {
		t.RequestID, _ = ctx.Value(logger.RequestIDKey).(string)
	}
	if t.TraceID == "" {
		t.TraceID, _ = ctx.Value(logger.TraceIDKey).(string)
	}
	if t.UserID == "" {
		t.UserID = database.UserIDFromContext(ctx)
	}
	if t.UserEmail == "" {
		if u, ok := ctx.Value(database.UserContextKey).(interface{ GetEmail() string }); ok {
			t.UserEmail = u.GetEmail()
		}
	}
}

func (r *Recorder) log(ctx context.Context, t *Trail) {
	if r.cfg.Logger == nil {
		return
	}

	fields := logrus.Fields{
		"audit_id":   t.ID,
		"audit_type": t.Type,
		"action":     t.Action,
		"entity":     t.Entity,
	}
	if t.EntityID != "" {
		fields["entity_id"] = t.EntityID
	}
	if t.UserID != "" {
		fields["user_id"] = t.UserID
	}
	if t.Type == TypeHTTP {
		fields["method"] = t.Method
		fields["path"] = t.Path
		fields["status"] = t.StatusCode
		fields["latency_ms"] = t.LatencyMs
		fields["ip"] = t.IP
	}

	r.cfg.Logger.WithCtx(ctx).WithFields(fields).Info("Audit trail")
}

func (r *Recorder) run() {
	defer close(r.done)

	ticker := time.NewTicker(r.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]*Trail, 0, r.cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		r.save(batch)
		batch = make([]*Trail, 0, r.cfg.BatchSize)
	}

	for {
		select {
		case t, ok := <-r.queue:
			if !ok {
				flush()
				return
			}
			batch = append(batch, t)
			if len(batch) >= r.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (r *Recorder) save(batch []*Trail) {
	if r.cfg.Store == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.SaveTimeout)
	defer cancel()

	if err := r.cfg.Store.Save(ctx, batch); err != nil && r.cfg.Logger != nil {
		r.cfg.Logger.WithError(err).WithField("count", len(batch)).Error("Failed to save audit trails")
	}
}
