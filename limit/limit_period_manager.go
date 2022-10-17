package limit

import (
	"context"
	"sync"
	"time"
)

const unsupportedPeriodLimitKind = "__unsupported_period_limit_kind__"

// PeriodLimitDriver driver interface
type PeriodLimitDriver interface {
	// Take requests a permit with context, it returns the permit state.
	Take(ctx context.Context, key string) (PeriodLimitState, error)
	// SetQuotaFull set a permit over quota.
	SetQuotaFull(ctx context.Context, key string) error
	// Del delete a permit
	Del(ctx context.Context, key string) error
	// TTL get key ttl
	// if key not exist, time = -1.
	// if key exist, but not set expire time, t = -2
	TTL(ctx context.Context, key string) (time.Duration, error)
	// GetInt get current count
	GetInt(ctx context.Context, key string) (int, bool, error)
}

// PeriodLimitManager manage limit period
type PeriodLimitManager struct {
	mu     sync.RWMutex
	driver map[string]PeriodLimitDriver
}

// NewPeriodLimitManager new a instance
func NewPeriodLimitManager() *PeriodLimitManager {
	return &PeriodLimitManager{
		driver: map[string]PeriodLimitDriver{
			unsupportedPeriodLimitKind: new(UnsupportedPeriodLimitDriver),
		},
	}
}

// NewPeriodLimitManagerWithDriver new a instance with driver
func NewPeriodLimitManagerWithDriver(drivers map[string]PeriodLimitDriver) *PeriodLimitManager {
	p := &PeriodLimitManager{
		driver: map[string]PeriodLimitDriver{
			unsupportedPeriodLimitKind: new(UnsupportedPeriodLimitDriver),
		},
	}
	for kind, drive := range drivers {
		p.driver[kind] = drive
	}
	return p
}

// Register register a PeriodLimitDriver with kind
func (p *PeriodLimitManager) Register(kind string, d PeriodLimitDriver) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.driver[kind]
	if ok {
		return ErrDuplicateDriver
	}
	p.driver[kind] = d
	return nil
}

// Acquire driver. if driver not exist. it will return UnsupportedPeriodLimitDriver.
func (p *PeriodLimitManager) Acquire(kind string) PeriodLimitDriver {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d, ok := p.driver[kind]
	if ok {
		return d
	}
	return p.driver[unsupportedPeriodLimitKind]
}

// UnsupportedPeriodLimitDriver unsupported limit period driver
type UnsupportedPeriodLimitDriver struct{}

func (u UnsupportedPeriodLimitDriver) Take(context.Context, string) (PeriodLimitState, error) {
	return PeriodLimitStsUnknown, ErrUnsupportedDriver
}
func (u UnsupportedPeriodLimitDriver) SetQuotaFull(context.Context, string) error {
	return ErrUnsupportedDriver
}
func (u UnsupportedPeriodLimitDriver) Del(context.Context, string) error {
	return ErrUnsupportedDriver
}
func (u UnsupportedPeriodLimitDriver) TTL(context.Context, string) (time.Duration, error) {
	return 0, ErrUnsupportedDriver
}
func (u UnsupportedPeriodLimitDriver) GetInt(context.Context, string) (int, bool, error) {
	return 0, false, ErrUnsupportedDriver
}
