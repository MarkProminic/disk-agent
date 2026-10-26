package configengine

import (
	"errors"

	"github.com/goccy/go-yaml"

	"github.com/MarkProminic/disk-agent/internal/problem"
	"github.com/MarkProminic/disk-agent/internal/safepath"
	"github.com/MarkProminic/disk-agent/internal/validation"
)

type prepared struct {
	name   string
	merged map[string]any
	filled map[string]any
	errors []problem.Error
}

func (e *Engine) prepare(name string, body map[string]any) prepared {
	schema := e.schemas[name]
	merged, _ := mergePatch(e.raw[name], body).(map[string]any)
	filled := fillDefaults(schema, merged)
	errs := []problem.Error{}
	for _, pointer := range readOnlyPointers(schema, body, "") {
		errs = append(errs, problem.Error{Pointer: pointer, Rule: "readOnly", Params: map[string]any{}})
	}
	errs = append(errs, validation.Object(schema, filled, schema)...)
	errs = append(errs, e.hookErrors(name, filled, body, e.hooks)...)
	return prepared{name: name, merged: merged, filled: filled, errors: errs}
}

func (e *Engine) writeFile(name string, merged map[string]any) error {
	target := e.filePath(name)
	if current, err := safepath.ReadFile(target); err == nil {
		if werr := safepath.WriteFile(target+".bak", current, 0o600); werr != nil {
			return werr
		}
	}
	raw, err := yaml.Marshal(merged)
	if err != nil {
		return err
	}
	return safepath.WriteFile(target, raw, 0o600)
}

// saved is one write the after-write function is told about once the lock is released.
type saved struct {
	name   string
	actor  string
	status RestartStatus
}

func (e *Engine) commit(p prepared, actor string) ([]RestartEntry, saved, error) {
	diff := restartDiff(e.schemas[p.name], e.filled[p.name], p.filled, "")
	if err := e.writeFile(p.name, p.merged); err != nil {
		return nil, saved{}, err
	}
	e.raw[p.name] = p.merged
	e.filled[p.name] = p.filled
	for _, entry := range diff {
		e.pending[entry.Pointer] = entry
	}
	who := actor
	when := now()
	e.lastModifiedBy = &who
	e.lastModifiedTime = &when
	return diff, saved{name: p.name, actor: actor, status: e.pendingStatus()}, nil
}

func (e *Engine) notifySaved(writes ...saved) {
	e.mu.RLock()
	hook := e.hooks.OnSaved
	e.mu.RUnlock()
	if hook == nil {
		return
	}
	for _, write := range writes {
		hook(write.name, write.actor, write.status)
	}
}

// Save applies a JSON Merge Patch to one file: validate, write atomically with a .bak, replace the cache, record the restart diff, then run the after-write function outside the lock.
func (e *Engine) Save(name string, body map[string]any, actor string) ([]RestartEntry, error) {
	e.mu.Lock()
	if !e.known(name) {
		e.mu.Unlock()
		return nil, errors.New("unknown configuration name " + name)
	}
	p := e.prepare(name, body)
	if len(p.errors) > 0 {
		e.mu.Unlock()
		return nil, &ValidationError{Errors: p.errors}
	}
	diff, write, err := e.commit(p, actor)
	e.mu.Unlock()
	if err != nil {
		return nil, err
	}
	e.notifySaved(write)
	return diff, nil
}

// Restore writes a whole raw document for one file after validating it, the backup-restore path.
func (e *Engine) Restore(name string, document map[string]any, actor string) error {
	e.mu.Lock()
	if !e.known(name) {
		e.mu.Unlock()
		return errors.New("unknown configuration name " + name)
	}
	schema := e.schemas[name]
	filled := fillDefaults(schema, document)
	errs := validation.Object(schema, filled, schema)
	errs = append(errs, e.hookErrors(name, filled, filled, e.hooks)...)
	if len(errs) > 0 {
		e.mu.Unlock()
		return &ValidationError{Errors: errs}
	}
	_, write, err := e.commit(prepared{name: name, merged: document, filled: filled}, actor)
	e.mu.Unlock()
	if err != nil {
		return err
	}
	e.notifySaved(write)
	return nil
}
