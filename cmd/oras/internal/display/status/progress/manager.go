/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package progress

import (
	"errors"
	"os"
	"sync"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/console"
	"oras.land/oras/internal/progress"
)

const (
	// bufferSize is the size of the status channel buffer.
	bufferSize       = 1
	framePerSecond   = 5
	bufFlushDuration = time.Second / framePerSecond
)

var errManagerStopped = errors.New("progress output manager has already been stopped")

type manager struct {
	status       []*status
	lock         sync.RWMutex // locks status and console, write lock is used for adding new status so that it has a higher priority
	console      console.Console
	updating     sync.WaitGroup
	renderDone   chan struct{}
	renderClosed chan struct{}
	prompts      map[progress.State]string
}

// NewManager initialized a new progress manager.
func NewManager(tty *os.File, prompts map[progress.State]string) (progress.Manager, error) {
	c, err := console.NewConsole(tty)
	if err != nil {
		return nil, err
	}
	return newManager(c, prompts), nil
}

func newManager(c console.Console, prompts map[progress.State]string) progress.Manager {
	m := &manager{
		console:      c,
		renderDone:   make(chan struct{}),
		renderClosed: make(chan struct{}),
		prompts:      prompts,
	}
	m.start()
	return m
}

func (m *manager) start() {
	m.console.Save()
	renderTicker := time.NewTicker(bufFlushDuration)
	go func() {
		defer m.console.Restore()
		defer renderTicker.Stop()
		for {
			select {
			case <-m.renderDone:
				m.render()
				close(m.renderClosed)
				return
			case <-renderTicker.C:
				m.render()
			}
		}
	}()
}

func (m *manager) render() {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// render with culling: only the latter statuses are rendered.
	models := m.status
	height, width := m.console.GetHeightWidth()
	if n := len(m.status) - height/2; n > 0 {
		models = models[n:]
		if height%2 == 1 {
			view := m.status[n-1].Render(width)
			m.console.OutputTo(uint(len(models)*2+1), view[1])
		}
	}
	viewHeight := len(models) * 2
	for i, model := range models {
		view := model.Render(width)
		m.console.OutputTo(uint(viewHeight-i*2), view[0])
		m.console.OutputTo(uint(viewHeight-i*2-1), view[1])
	}
}

// Track appends a new status with 2-line space for rendering.
func (m *manager) Track(desc ocispec.Descriptor) (progress.Tracker, error) {
	if m.closed() {
		return nil, errManagerStopped
	}

	m.render()
	s := newStatus(desc)
	m.lock.Lock()
	m.status = append(m.status, s)
	m.console.NewRow()
	m.console.NewRow()
	m.lock.Unlock()
	return m.newTracker(s), nil
}

func (m *manager) newTracker(s *status) progress.Tracker {
	ch := make(chan statusUpdate, bufferSize)
	m.updating.Add(1)
	go func() {
		defer m.updating.Done()
		for update := range ch {
			update(s)
		}
	}()
	return &messenger{
		update:  ch,
		prompts: m.prompts,
	}
}

// Close stops all status and waits for updating and rendering.
func (m *manager) Close() error {
	if m.closed() {
		return errManagerStopped
	}
	// 1. wait for update to stop
	m.updating.Wait()
	// 2. stop periodic rendering
	close(m.renderDone)
	// 3. wait for the render stop
	<-m.renderClosed
	return nil
}

func (m *manager) closed() bool {
	select {
	case <-m.renderClosed:
		return true
	default:
		return false
	}
}
