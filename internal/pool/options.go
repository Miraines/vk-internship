package pool

import (
	"context"
	"fmt"
	"log"
)

type Option func(*options)

type options struct {
	initialWorkers int
	logger         *log.Logger

	userHandler Handler
	middlewares []Middleware

	autoScale autoScaleConfig
	rateRPS   int
}

func defaultOptions() *options {
	return &options{
		initialWorkers: 1,
		logger:         log.Default(),
		userHandler: func(ctx context.Context, job string) {
			fmt.Println(job)
		},
		middlewares: nil,
		autoScale:   autoScaleConfig{},
		rateRPS:     0,
	}
}

func WithInitialWorkers(n int) Option {
	return func(o *options) { o.initialWorkers = n }
}

func WithLogger(l *log.Logger) Option {
	return func(o *options) { o.logger = l }
}

func WithUserHandler(h Handler) Option {
	return func(o *options) { o.userHandler = h }
}

func WithMiddleware(mw ...Middleware) Option {
	return func(o *options) { o.middlewares = append(o.middlewares, mw...) }
}

func WithAutoScaler(cfg AutoScaleConfig) Option {
	return func(o *options) { o.autoScale = autoScaleConfig{enabled: true, AutoScaleConfig: cfg} }
}

func WithRateLimit(rps int) Option {
	return func(o *options) { o.rateRPS = rps }
}

func (o *options) composeMiddleware(h Handler) Handler {
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		h = o.middlewares[i](h)
	}
	if o.rateRPS > 0 {
		h = RateLimitMiddleware(o.rateRPS)(h)
	}
	return h
}
