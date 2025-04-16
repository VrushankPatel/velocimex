package strategy

import (
	"context"
)

// GetAllResults returns results for all strategies
func (e *Engine) GetAllResults() map[string]StrategyResults {
	e.mu.RLock()
	defer e.mu.RUnlock()

	results := make(map[string]StrategyResults)
	for name, strategy := range e.strategies {
		results[name] = strategy.GetResults()
	}

	return results
}

// StartAll starts all registered strategies
func (e *Engine) StartAll(ctx context.Context) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, strategy := range e.strategies {
		if err := strategy.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

// StopAll stops all registered strategies
func (e *Engine) StopAll() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, strategy := range e.strategies {
		if err := strategy.Stop(); err != nil {
			return err
		}
	}

	return nil
}

// UnregisterStrategy removes a strategy from the engine
func (e *Engine) UnregisterStrategy(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.strategies, name)
}

// GetStrategy returns a strategy by name
func (e *Engine) GetStrategy(name string) (Strategy, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	strategy, exists := e.strategies[name]
	return strategy, exists
}

// GetAllStrategies returns all registered strategies
func (e *Engine) GetAllStrategies() []Strategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]Strategy, 0, len(e.strategies))
	for _, strategy := range e.strategies {
		result = append(result, strategy)
	}

	return result
}
