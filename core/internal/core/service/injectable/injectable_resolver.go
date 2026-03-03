package injectable

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

const (
	// DefaultInjectorTimeout is the default timeout for injectors.
	DefaultInjectorTimeout = 30 * time.Second
)

// ResolveResult contains the results of injector resolution.
type ResolveResult struct {
	mu sync.Mutex

	// Values contains the resolved values (code -> value).
	Values map[string]entity.InjectableValue

	// Errors contains errors from non-critical injectors.
	Errors map[string]error

	// Metadata contains additional metadata per injector.
	Metadata map[string]map[string]any
}

// InjectableResolverService resolves injector values.
type InjectableResolverService struct {
	registry port.InjectorRegistry
}

// NewInjectableResolverService creates a new resolution service.
func NewInjectableResolverService(registry port.InjectorRegistry) *InjectableResolverService {
	return &InjectableResolverService{
		registry: registry,
	}
}

// Resolve resolves the values of the referenced injectors.
// Executes Init() GLOBAL first, then resolves injectors by dependency levels.
// Codes not found in the registry are delegated to the WorkspaceInjectableProvider if one is registered.
func (s *InjectableResolverService) Resolve(
	ctx context.Context,
	injCtx *entity.InjectorContext,
	referencedCodes []string,
) (*ResolveResult, error) {
	result := &ResolveResult{
		Values:   make(map[string]entity.InjectableValue),
		Errors:   make(map[string]error),
		Metadata: make(map[string]map[string]any),
	}

	if len(referencedCodes) == 0 {
		return result, nil
	}

	// Partition codes into registry codes and provider codes
	registryCodes, providerCodes := s.partitionCodes(referencedCodes)

	// 1. Execute Init() GLOBAL if defined
	initFunc := s.registry.GetInitFunc()
	if initFunc != nil {
		slog.DebugContext(ctx, "executing global init function")
		initData, err := initFunc(ctx, injCtx)
		if err != nil {
			return nil, fmt.Errorf("global init failed: %w", err)
		}
		injCtx.SetInitData(initData)
	}

	// 2. Resolve registry injectors via dependency graph
	if len(registryCodes) > 0 {
		if err := s.resolveRegistryCodes(ctx, injCtx, registryCodes, result); err != nil {
			return nil, err
		}
	}

	// 3. Resolve provider codes via WorkspaceInjectableProvider
	if len(providerCodes) > 0 {
		if err := s.resolveProviderCodes(ctx, injCtx, providerCodes, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// partitionCodes separates codes into registry-known and provider-bound codes.
func (s *InjectableResolverService) partitionCodes(codes []string) (registryCodes, providerCodes []string) {
	provider := s.registry.GetWorkspaceInjectableProvider()

	for _, code := range codes {
		if _, ok := s.registry.Get(code); ok {
			registryCodes = append(registryCodes, code)
		} else if provider != nil {
			providerCodes = append(providerCodes, code)
		}
		// If code is not in registry and no provider, it will be silently skipped
		// (same behavior as before — the dependency graph builder skips unknown codes)
	}

	return registryCodes, providerCodes
}

// resolveRegistryCodes resolves codes through the injector registry using the dependency graph.
func (s *InjectableResolverService) resolveRegistryCodes(
	ctx context.Context,
	injCtx *entity.InjectorContext,
	codes []string,
	result *ResolveResult,
) error {
	// Build dependency graph
	graph := NewDependencyGraph()
	err := graph.BuildFromInjectors(
		func(code string) ([]string, bool) {
			inj, ok := s.registry.Get(code)
			if !ok {
				return nil, false
			}
			_, deps := inj.Resolve()
			return deps, true
		},
		codes,
	)
	if err != nil {
		return fmt.Errorf("building dependency graph: %w", err)
	}

	// Get execution order (by levels)
	levels, err := graph.TopologicalSort()
	if err != nil {
		return fmt.Errorf("topological sort: %w", err)
	}

	// Execute injectors by levels
	for levelIdx, level := range levels {
		slog.DebugContext(ctx, "executing injector level",
			"level", levelIdx,
			"injectors", level,
		)

		if err := s.executeLevel(ctx, injCtx, level, result); err != nil {
			return err
		}
	}

	return nil
}

// resolveProviderCodes delegates resolution of provider-bound codes to the WorkspaceInjectableProvider.
func (s *InjectableResolverService) resolveProviderCodes(
	ctx context.Context,
	injCtx *entity.InjectorContext,
	codes []string,
	result *ResolveResult,
) error {
	provider := s.registry.GetWorkspaceInjectableProvider()
	if provider == nil {
		return nil
	}

	slog.DebugContext(ctx, "resolving provider codes", "codes", codes)

	req := &port.ResolveInjectablesRequest{
		TenantCode:      injCtx.TenantCode(),
		WorkspaceCode:   injCtx.WorkspaceCode(),
		TemplateID:      injCtx.TemplateID(),
		Codes:           codes,
		SelectedFormats: injCtx.GetSelectedFormats(),
		Headers:         injCtx.GetHeaders(),
		Payload:         injCtx.RequestPayload(),
		InitData:        injCtx.InitData(),
		Environment:     injCtx.Environment(),
	}

	providerResult, err := provider.ResolveInjectables(ctx, req)
	if err != nil {
		return fmt.Errorf("workspace injectable provider resolution failed: %w", err)
	}

	if providerResult == nil {
		return nil
	}

	// Merge provider values into result
	for code, val := range providerResult.Values {
		if val != nil {
			result.Values[code] = *val
			injCtx.SetResolved(code, val.AsAny())
		}
	}

	// Merge provider errors as non-critical
	for code, errMsg := range providerResult.Errors {
		result.Errors[code] = fmt.Errorf("%s", errMsg)
	}

	slog.DebugContext(ctx, "provider codes resolved",
		"resolved", len(providerResult.Values),
		"errors", len(providerResult.Errors),
	)

	return nil
}

// executeLevel executes all injectors in a level in parallel.
func (s *InjectableResolverService) executeLevel(
	ctx context.Context,
	injCtx *entity.InjectorContext,
	codes []string,
	result *ResolveResult,
) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, code := range codes {
		g.Go(func() error {
			return s.executeInjector(gCtx, injCtx, code, result)
		})
	}

	return g.Wait()
}

// executeInjector executes an individual injector.
func (s *InjectableResolverService) executeInjector(
	ctx context.Context,
	injCtx *entity.InjectorContext,
	code string,
	result *ResolveResult,
) error {
	inj, ok := s.registry.Get(code)
	if !ok {
		slog.WarnContext(ctx, "injector not found", "code", code)
		return nil
	}

	// Get the resolution function
	resolveFunc, _ := inj.Resolve()
	if resolveFunc == nil {
		slog.WarnContext(ctx, "injector has nil resolve function", "code", code)
		return nil
	}

	// Determine timeout
	timeout := inj.Timeout()
	if timeout <= 0 {
		timeout = DefaultInjectorTimeout
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute injector
	slog.DebugContext(ctx, "executing injector", "code", code, "timeout", timeout)

	injResult, err := resolveFunc(timeoutCtx, injCtx)
	if err != nil {
		slog.ErrorContext(ctx, "injector failed",
			"code", code,
			"error", err,
			"critical", inj.IsCritical(),
		)

		if inj.IsCritical() {
			return fmt.Errorf("critical injector %q failed: %w", code, err)
		}

		// Non-critical error, save and continue
		result.mu.Lock()
		result.Errors[code] = err
		result.mu.Unlock()
		return nil
	}

	// Save result
	if injResult != nil {
		result.mu.Lock()
		result.Values[code] = injResult.Value
		if injResult.Metadata != nil {
			result.Metadata[code] = injResult.Metadata
		}
		result.mu.Unlock()

		injCtx.SetResolved(code, injResult.Value.AsAny())
	}

	slog.DebugContext(ctx, "injector completed", "code", code)
	return nil
}

// MergeWithPayloadValues combines injector values with values extracted from the payload.
// Payload values have priority (they overwrite injector values).
func (s *InjectableResolverService) MergeWithPayloadValues(
	resolved *ResolveResult,
	payloadValues map[string]entity.InjectableValue,
) map[string]any {
	merged := make(map[string]any)

	// First add injector values
	for code, value := range resolved.Values {
		merged[code] = value.AsAny()
	}

	// Then overwrite with payload values
	for key, value := range payloadValues {
		merged[key] = value.AsAny()
	}

	return merged
}
