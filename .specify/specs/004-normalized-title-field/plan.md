# Implementation Plan: Normalized Title Field

## Feature Overview
Add a `normalized_title` field to track scrobbles that removes common annotations (Live, Remastered, featuring, etc.) from track titles for improved matching and grouping.

## Planning Artifacts
- **Spec**: [spec.md](spec.md) - Complete feature specification
- **Research**: [research.md](research.md) - Pattern analysis and technical decisions
- **Data Model**: [data-model.md](data-model.md) - Structures and interfaces
- **Contracts**: [contracts/](contracts/) - JSON schemas and examples
- **Quickstart**: [quickstart.md](quickstart.md) - User-facing documentation
- **Constitution Check**: [constitution-check.md](constitution-check.md) - Compliance verification

## Implementation Phases

### Phase 0: Foundation ✅ COMPLETE
Research and design complete. All planning artifacts created.

### Phase 1: Core Implementation 🎯 NEXT
Implement the normalization package with pattern matching and configuration.

**Priority**: P1 (Required for feature functionality)

**Tasks**: See [tasks.md](tasks.md) for detailed breakdown

**Estimated Effort**: 8-12 hours

**Deliverables**:
- `internal/normalize/normalize.go` - Core normalization logic
- `internal/normalize/patterns.go` - Pattern definitions
- `internal/normalize/config.go` - Configuration loading
- `internal/normalize/normalize_test.go` - Comprehensive test suite

### Phase 2: Integration
Integrate normalization into scrobble data flow.

**Priority**: P1 (Required for feature functionality)

**Estimated Effort**: 4-6 hours

**Deliverables**:
- Updated `internal/models/scrobble.go` with `NormalizedTitle` field
- Modified `NewScrobble()` constructor to populate normalized title
- Updated JSON marshaling
- Integration tests

### Phase 3: Configuration
Add configuration support for customization.

**Priority**: P2 (Enhanced functionality)

**Estimated Effort**: 3-4 hours

**Deliverables**:
- YAML configuration file support
- Environment variable mapping
- Pattern enable/disable
- Custom pattern support

### Phase 4: Documentation & Polish
Complete documentation and operational readiness.

**Priority**: P2 (Production readiness)

**Estimated Effort**: 2-3 hours

**Deliverables**:
- Updated `docs/configuration.md`
- Example configuration files
- Troubleshooting guide additions
- Updated README.md

## Success Criteria

### Functional
- ✅ 95%+ of titles correctly normalized
- ✅ <1% false positive rate
- ✅ Original title always preserved
- ✅ Backward compatible JSON output

### Quality
- ✅ >90% test coverage
- ✅ All edge cases tested
- ✅ Cyclomatic complexity <10 per function
- ✅ Passes all linting checks

### Performance
- ✅ <1ms per title normalization
- ✅ No impact on API response time
- ✅ Memory overhead <10%
- ✅ Benchmarks pass

### Operational
- ✅ Feature flag for enable/disable
- ✅ DEBUG logging when titles change
- ✅ Configuration validated at startup
- ✅ Graceful error handling

## Dependencies

### External Libraries
- `gopkg.in/yaml.v3` - YAML configuration parsing

### Internal Dependencies
- `internal/models` - Scrobble struct definition
- `internal/config` - Configuration system
- `internal/logging` - Logging infrastructure

### Tooling
- Go 1.24.0+
- Existing test framework
- golint, gofmt (existing)

## Risk Management

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Over-normalization | High | Medium | Extensive test suite, min length check, original preserved |
| Performance impact | Medium | Low | Pre-compiled patterns, benchmarks enforced |
| False positives | Medium | Medium | End-anchored patterns, debug logging, feature flag |
| Configuration errors | Low | Low | Validation at load, safe defaults |

## Rollout Strategy

### Phase 1: Development
1. Implement core normalization (internal/normalize)
2. Unit test to >90% coverage
3. Integration test with sample data

### Phase 2: Testing
1. Test with real Last.fm data samples
2. Validate performance benchmarks
3. Review false positive rate
4. Adjust patterns if needed

### Phase 3: Deployment
1. Feature flag: `NORMALIZE_ENABLED=true` (default)
2. Deploy to production
3. Monitor DEBUG logs for normalization activity
4. Collect feedback on false positives

### Phase 4: Refinement
1. Add custom patterns based on feedback
2. Tune min_length threshold if needed
3. Consider caching if performance issues arise

## Monitoring Plan

### Initial Deployment
- **Week 1-2**: DEBUG logging enabled for all normalizations
- **Metrics**: Track titles normalized vs. unchanged ratio
- **Review**: Manually review sample of normalized titles for accuracy

### Steady State
- **Logging**: DEBUG level only (operator can enable)
- **Metrics**: Pattern match frequency (future enhancement)
- **Alerts**: Configuration errors logged at ERROR level

## Rollback Plan

### Immediate Rollback
```bash
# Disable feature globally
export NORMALIZE_ENABLED=false
# Restart service
```

### Partial Rollback
```yaml
# Disable specific problematic patterns
normalize:
  enabled: true
  disabled_patterns:
    - live  # Example: if "live" pattern causing issues
```

### Full Removal
- Feature flag prevents execution
- Original `track` field unchanged
- No data migration needed (additive field)

## Timeline

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Phase 1: Core Implementation | 8-12 hours | None |
| Phase 2: Integration | 4-6 hours | Phase 1 complete |
| Phase 3: Configuration | 3-4 hours | Phase 2 complete |
| Phase 4: Documentation | 2-3 hours | Phase 3 complete |
| **Total** | **17-25 hours** | - |

**Note**: Timeline assumes single developer, full-time focus. Actual duration may vary based on review cycles and testing iterations.

## Definition of Done

### Code Complete
- [ ] All functions implemented
- [ ] >90% test coverage achieved
- [ ] All tests passing
- [ ] Benchmarks meet <1ms target
- [ ] No linting errors

### Integration Complete
- [ ] Scrobble struct updated
- [ ] NewScrobble() constructor modified
- [ ] JSON output includes normalized_title
- [ ] Integration tests passing

### Documentation Complete
- [ ] Quickstart guide reviewed
- [ ] Configuration documented
- [ ] Code comments complete
- [ ] Examples tested

### Production Ready
- [ ] Constitution check passed
- [ ] Code review approved
- [ ] Performance benchmarks validated
- [ ] Feature flag operational
- [ ] Rollback plan tested

## Next Steps

1. Review this plan with stakeholders
2. Begin Phase 1: Core Implementation
3. Follow TDD discipline (tests before code)
4. Regular check-ins at phase boundaries

## References

- [Feature Spec](spec.md)
- [Research Document](research.md)
- [Data Model](data-model.md)
- [Constitution](../../.specify/memory/constitution.md)
- [Task Breakdown](tasks.md)
